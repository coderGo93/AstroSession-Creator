package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if (strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'")) || (strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"")) {
		p = p[1 : len(p)-1]
	}
	p = strings.ReplaceAll(p, "\\ ", " ")
	return p
}

func calculateTotalSize(src string) int64 {
	var size int64
	info, err := os.Stat(src)
	if err != nil {
		return 0
	}
	if info.IsDir() {
		entries, _ := os.ReadDir(src)
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				fileInfo, err := e.Info()
				if err == nil {
					size += fileInfo.Size()
				}
			}
		}
	} else {
		size += info.Size()
	}
	return size
}

func printProgressBar(totalBytes *int64, movedBytes *int64, done chan bool) {
	startTime := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			currentTotal := atomic.LoadInt64(totalBytes)
			currentMoved := atomic.LoadInt64(movedBytes)

			if currentTotal == 0 {
				continue
			}

			percent := float64(currentMoved) / float64(currentTotal) * 100
			if percent > 100 {
				percent = 100
			}

			barLength := 50
			filledLen := int(percent / 100 * float64(barLength))
			bar := strings.Repeat("=", filledLen) + strings.Repeat("-", barLength-filledLen)

			elapsedTime := time.Since(startTime).Seconds()
			speed := float64(currentMoved) / elapsedTime // bytes per second
			eta := "N/A"

			if speed > 0 {
				remainingBytes := float64(currentTotal - currentMoved)
				remainingSeconds := remainingBytes / speed
				if remainingSeconds < 60 {
					eta = fmt.Sprintf("%.0fs", remainingSeconds)
				} else {
					eta = fmt.Sprintf("%.1fm", remainingSeconds/60)
				}
			}

			fmt.Printf("\rProgreso: [%s] %.1f%% | ETA: %-6s", bar, percent, eta)
		}
	}
}

func moveCrossDevice(src, dst string, movedBytes *int64) error {
	info, err := os.Stat(src)
	fileSize := int64(0)
	if err == nil {
		fileSize = info.Size()
	}

	err = os.Rename(src, dst)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "cross-device") || strings.Contains(strings.ToLower(err.Error()), "invalid cross-device") || runtime.GOOS == "windows" {
			return copyAndDelete(src, dst, movedBytes)
		}
		return err
	}

	// Si Rename funcionó rápido (mismo dispositivo), sumamos los bytes de golpe
	if movedBytes != nil && fileSize > 0 {
		atomic.AddInt64(movedBytes, fileSize)
	}
	return nil
}

func copyAndDelete(src, dst string, movedBytes *int64) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 32*1024) // 32KB buffer (tamaño estándar en go io.Copy)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			if movedBytes != nil {
				atomic.AddInt64(movedBytes, int64(n))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	in.Close() // Cerrar el origen antes de borrarlo
	return os.Remove(src)
}

func checkDuplicates(srcClean, destDir string) bool {
	info, err := os.Stat(srcClean)
	if err != nil {
		return false
	}
	if info.IsDir() {
		entries, err := os.ReadDir(srcClean)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					destPath := filepath.Join(destDir, e.Name())
					if _, err := os.Stat(destPath); err == nil {
						return true
					}
				}
			}
		}
	} else {
		destPath := filepath.Join(destDir, filepath.Base(srcClean))
		if _, err := os.Stat(destPath); err == nil {
			return true
		}
	}
	return false
}

func getUniqueDestPath(destPath string) string {
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return destPath
	}

	ext := filepath.Ext(destPath)
	base := strings.TrimSuffix(destPath, ext)
	counter := 1

	for {
		newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

func moveFiles(srcClean, destDir string, wg *sync.WaitGroup, movedBytes *int64) {
	defer wg.Done()

	info, err := os.Stat(srcClean)
	if err != nil {
		fmt.Printf("❌ Error al leer origen '%s': %v\n", srcClean, err)
		return
	}

	count := 0
	if info.IsDir() {
		entries, err := os.ReadDir(srcClean)
		if err != nil {
			fmt.Printf("❌ Error leyendo %s: %v\n", srcClean, err)
			return
		}
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				srcPath := filepath.Join(srcClean, e.Name())
				destPath := getUniqueDestPath(filepath.Join(destDir, e.Name()))

				if err := moveCrossDevice(srcPath, destPath, movedBytes); err != nil {
					fmt.Printf("\n  Error moviendo %s: %v\n", e.Name(), err)
				} else {
					count++
				}
			}
		}
	} else {
		destPath := getUniqueDestPath(filepath.Join(destDir, filepath.Base(srcClean)))
		if err := moveCrossDevice(srcClean, destPath, movedBytes); err != nil {
			fmt.Printf("\n  Error moviendo %s: %v\n", info.Name(), err)
		} else {
			count++
		}
	}
	fmt.Printf("\n✅ %d archivos movidos exitosamente a -> %s\n", count, filepath.Base(destDir))
}
