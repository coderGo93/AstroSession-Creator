package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// readInput cleans terminal inputs on both Windows (\r\n) and Linux/Mac (\n)
func readInput(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	if runtime.GOOS == "windows" {
		input = strings.TrimSuffix(input, "\r\n")
	} else {
		input = strings.TrimSuffix(input, "\n")
	}
	return strings.TrimSpace(input)
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("==============================================")
	fmt.Println("=== Astrophotography Session Creator ===")
	fmt.Println("==============================================")

	fmt.Print("\nCaptured object name (e.g. M81, M81 M82, NGC 4236): ")
	targetInput := readInput(reader)

	if targetInput == "" {
		fmt.Println("You must enter a valid name.")
		return
	}

	targets := strings.Fields(targetInput)
	var resolvedTechNames []string
	var commonNames []string
	allHaveCommonName := true

	fmt.Printf("\nSearching for information on '%s' in SIMBAD/Sesame...\n", targetInput)

	for _, t := range targets {
		formatted := formatTargetName(t)
		cName, tOptions := querySesame(t)

		techName := formatted
		if len(tOptions) > 0 {
			if len(tOptions) == 1 {
				techName = tOptions[0]
				fmt.Printf("-> [%s] Using primary technical designation: %s\n", t, techName)
			} else {
				fmt.Printf("\nMultiple catalog designations found for [%s]:\n", t)
				for i, opt := range tOptions {
					fmt.Printf("  %d) %s\n", i+1, opt)
				}
				fmt.Printf("  %d) Keep original: %s\n", len(tOptions)+1, formatted)
				fmt.Printf("Which nomenclature do you prefer for the main folder? (1-%d) [1]: ", len(tOptions)+1)

				optInput := readInput(reader)
				if optInput == "" {
					optInput = "1"
				}
				idx, err := strconv.Atoi(optInput)
				if err == nil && idx >= 1 && idx <= len(tOptions) {
					techName = tOptions[idx-1]
				}
			}
		} else if cName == "" {
			fmt.Printf("-> Object [%s] not found or without common name (only using '%s').\n", t, formatted)
		}

		resolvedTechNames = append(resolvedTechNames, techName)

		if cName != "" {
			commonNames = append(commonNames, cName)
		} else {
			allHaveCommonName = false
		}
	}

	finalTargetFolder := strings.Join(resolvedTechNames, "_")

	if len(commonNames) > 0 && allHaveCommonName {
		joinedCommon := strings.Join(commonNames, " & ")
		if len(commonNames) > 1 && !strings.HasSuffix(strings.ToLower(joinedCommon), "galaxies") {
			// small grammatical touch for pluralizing multiple known galaxies if they aren't labeled already
			if strings.HasSuffix(strings.ToLower(joinedCommon), "galaxy") {
				joinedCommon = strings.TrimSuffix(joinedCommon, "Galaxy") + "Galaxies"
			}
		}
		finalTargetFolder = fmt.Sprintf("%s (%s)", finalTargetFolder, joinedCommon)
	} else if len(commonNames) > 0 && len(targets) == 1 {
		// Only 1 target input, perfectly append its common name.
		finalTargetFolder = fmt.Sprintf("%s (%s)", finalTargetFolder, commonNames[0])
	} // For multiple where 1 fails, revert strict to technical

	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	executableDir := filepath.Dir(os.Args[0])
	if filepath.IsAbs(executableDir) {
		baseDir = executableDir
	}

	normFinal := normalizeName(finalTargetFolder)
	normTarget := normalizeName(targetInput)

	var similarFolder string
	filesInfo, err := os.ReadDir(baseDir)
	if err == nil {
		for _, info := range filesInfo {
			if info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
				normFolder := normalizeName(info.Name())
				if normFolder == normFinal || normFolder == normTarget {
					similarFolder = info.Name()
					break
				}
			}
		}
	}

	if similarFolder != "" && similarFolder != finalTargetFolder {
		fmt.Println()
		fmt.Printf("⚠️  An existing very similar folder was found: '%s'\n", similarFolder)
		fmt.Printf("    The new standardized format is: '%s'\n", finalTargetFolder)
		fmt.Println("\nWhat do you want to do?")
		fmt.Println("  1) Use the existing folder as is and add the new session inside.")
		fmt.Printf("  2) Rename the existing folder to '%s' and add the new session there.\n", finalTargetFolder)
		fmt.Printf("  3) Ignore and create '%s' as a completely new folder.\n", finalTargetFolder)

		for {
			fmt.Print("Choose an option (1/2/3) [1]: ")
			resp := readInput(reader)

			if resp == "" || resp == "1" {
				finalTargetFolder = similarFolder
				fmt.Printf("-> We will operate inside: '%s'\n", finalTargetFolder)
				break
			} else if resp == "2" {
				oldPath := filepath.Join(baseDir, similarFolder)
				newPath := filepath.Join(baseDir, finalTargetFolder)
				err := os.Rename(oldPath, newPath)
				if err != nil {
					fmt.Printf("-> Error renaming the folder: %v\n", err)
					fmt.Println("-> We will operate with the original name for safety.")
					finalTargetFolder = similarFolder
				} else {
					fmt.Printf("-> Folder successfully renamed to '%s'!\n", finalTargetFolder)
				}
				break
			} else if resp == "3" {
				fmt.Printf("-> We will create a new folder: '%s'\n", finalTargetFolder)
				break
			} else {
				fmt.Println("Invalid option.")
			}
		}
	}

	fmt.Println("\n----------------------------------------------")
	fmt.Println("Enter the capture date. Options:")
	ahora := time.Now()
	hoyStr := fmt.Sprintf("%d %s", ahora.Day(), monthNames[int(ahora.Month())])
	esteAñoStr := fmt.Sprintf("%d", ahora.Year())

	fmt.Printf(" [Empty ENTER] -> Use today: %s (Year: %s)\n", hoyStr, esteAñoStr)
	fmt.Printf(" '12 feb'      -> Use this date (Year: %s)\n", esteAñoStr)
	fmt.Println(" '12 feb 2025' -> Use this date and year")
	fmt.Print("Date: ")

	finalDate := readInput(reader)
	finalYear := esteAñoStr

	if finalDate == "" {
		finalDate = hoyStr
	} else {
		parts := strings.Split(finalDate, " ")
		if len(parts) == 3 {
			finalDate = fmt.Sprintf("%s %s", parts[0], parts[1])
			if match, _ := regexp.MatchString(`^\d{4}$`, parts[2]); match {
				finalYear = parts[2]
			}
		}
	}

	targetPath := filepath.Join(baseDir, finalTargetFolder, finalYear, finalDate)

	if _, err := os.Stat(targetPath); err == nil {
		hasFiles := false
		err = filepath.WalkDir(targetPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
				hasFiles = true
				return filepath.SkipDir // detiene el escaneo
			}
			return nil
		})

		if hasFiles {
			fmt.Println("\n" + strings.Repeat("=", 50))
			fmt.Println("🚨 WARNING: EXISTING SESSION DETECTED 🚨")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Printf("A capture for '%s' already exists on %s %s.\n", finalTargetFolder, finalDate, finalYear)
			fmt.Printf("Path: %s\n", targetPath)
			fmt.Println("In addition, the folder ALREADY CONTAINS FILES inside (photos, logs, etc).")
			fmt.Println("Taking the same object, 2 times, on the exact same day is unusual.")

			fmt.Print("\nAre you sure you want to mix new sessions on this date? (y/n) [n]: ")
			resp := strings.ToLower(readInput(reader))
			if resp != "y" {
				fmt.Println("Operation canceled. (No folder was created or modified).")
				fmt.Println("\nPress Enter to exit...")
				readInput(reader)
				return
			}
		}
	}

	for _, folder := range subfolders {
		folderPath := filepath.Join(targetPath, filepath.FromSlash(folder))
		err := os.MkdirAll(folderPath, 0755)
		if err != nil {
			fmt.Printf("❌ Error creating subfolder %s: %v\n", folder, err)
			return
		}
	}

	fmt.Println("\n✅ Structure successfully generated!")
	fmt.Printf("📁 Path created: %s\n", targetPath)
	fmt.Printf("📂 Subfolders generated: %s\n", strings.Join(subfolders, ", "))

	fmt.Print("\nDo you want to MOVE your photos (Lights/Flats) to these new folders? (y/n) [n]: ")
	respMove := strings.ToLower(readInput(reader))
	if respMove == "y" {
		fmt.Print("\nDrag your Lights FOLDER here (or leave empty to skip): ")
		lightsSrc := cleanPath(readInput(reader))

		fmt.Print("Drag your Flats FOLDER here (or leave empty to skip): ")
		flatsSrc := cleanPath(readInput(reader))

		if lightsSrc != "" || flatsSrc != "" {
			hasDuplicates := false
			if lightsSrc != "" && checkDuplicates(lightsSrc, filepath.Join(targetPath, "Lights")) {
				hasDuplicates = true
			}
			if flatsSrc != "" && checkDuplicates(flatsSrc, filepath.Join(targetPath, "Flats")) {
				hasDuplicates = true
			}

			if hasDuplicates {
				fmt.Print("\n⚠️  WARNING: Possible duplicates detected in destination. They will be renamed by appending _1, _2... Do you wish to continue and duplicate them? (y/n) [n]: ")
				respDup := strings.ToLower(readInput(reader))
				if respDup != "y" {
					fmt.Println("File move operation canceled.")
					goto END_MOVE
				}
			}

			fmt.Println("\nPreparing files to move...")
			var totalBytes int64
			var movedBytes int64

			if lightsSrc != "" {
				totalBytes += calculateTotalSize(lightsSrc)
			}
			if flatsSrc != "" {
				totalBytes += calculateTotalSize(flatsSrc)
			}

			fmt.Println("Starting transfer...")
			var wg sync.WaitGroup

			doneChan := make(chan bool)
			go printProgressBar(&totalBytes, &movedBytes, doneChan)

			if lightsSrc != "" {
				wg.Add(1)
				go moveFiles(lightsSrc, filepath.Join(targetPath, "Lights"), &wg, &movedBytes)
			}
			if flatsSrc != "" {
				wg.Add(1)
				go moveFiles(flatsSrc, filepath.Join(targetPath, "Flats"), &wg, &movedBytes)
			}

			wg.Wait()
			doneChan <- true

			fmt.Printf("\rProgress: [==================================================] 100%% | ETA: 0s          \n")
			fmt.Println("\nMove process completed!")
		}
	}

END_MOVE:
	fmt.Println("\nPress Enter to exit...")
	readInput(reader)
}
