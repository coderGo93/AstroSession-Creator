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

// readInput lee limpia terminal inputs tanto en Windows (\r\n) como Linux/Mac (\n)
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
	fmt.Println("=== Creador de Sesiones de Astrofotografía ===")
	fmt.Println("==============================================")

	fmt.Print("\nNombre del objeto capturado (ej. M81, M81 M82, NGC 4236): ")
	targetInput := readInput(reader)

	if targetInput == "" {
		fmt.Println("Debes ingresar un nombre válido.")
		return
	}

	targets := strings.Fields(targetInput)
	var resolvedTechNames []string
	var commonNames []string
	allHaveCommonName := true

	fmt.Printf("\nBuscando información para '%s' en SIMBAD/Sesame...\n", targetInput)

	for _, t := range targets {
		formatted := formatTargetName(t)
		cName, tOptions := querySesame(t)

		techName := formatted
		if len(tOptions) > 0 {
			if len(tOptions) == 1 {
				techName = tOptions[0]
				fmt.Printf("-> [%s] Usando designación técnica principal: %s\n", t, techName)
			} else {
				fmt.Printf("\nSe encontraron múltiples designaciones de catálogo para [%s]:\n", t)
				for i, opt := range tOptions {
					fmt.Printf("  %d) %s\n", i+1, opt)
				}
				fmt.Printf("  %d) Mantener original: %s\n", len(tOptions)+1, formatted)
				fmt.Printf("¿Qué nomenclatura prefieres usar para la carpeta principal? (1-%d) [1]: ", len(tOptions)+1)

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
			fmt.Printf("-> Objeto [%s] no encontrado o sin nombre común (sólo usaré '%s').\n", t, formatted)
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
		fmt.Println("Error obteniendo el directorio actual:", err)
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
		fmt.Printf("⚠️  Se encontró una carpeta existente muy similar: '%s'\n", similarFolder)
		fmt.Printf("    El nuevo formato estandarizado es: '%s'\n", finalTargetFolder)
		fmt.Println("\n¿Qué deseas hacer?")
		fmt.Println("  1) Usar la carpeta existente tal como está y agregar la nueva sesión ahí dentro.")
		fmt.Printf("  2) Renombrar la carpeta existente a '%s' y agregar la nueva sesión ahí.\n", finalTargetFolder)
		fmt.Printf("  3) Ignorar y crear '%s' como una carpeta completamente nueva.\n", finalTargetFolder)

		for {
			fmt.Print("Elige una opción (1/2/3) [1]: ")
			resp := readInput(reader)

			if resp == "" || resp == "1" {
				finalTargetFolder = similarFolder
				fmt.Printf("-> Operaremos dentro de: '%s'\n", finalTargetFolder)
				break
			} else if resp == "2" {
				oldPath := filepath.Join(baseDir, similarFolder)
				newPath := filepath.Join(baseDir, finalTargetFolder)
				err := os.Rename(oldPath, newPath)
				if err != nil {
					fmt.Printf("-> Error al renombrar la carpeta: %v\n", err)
					fmt.Println("-> Operaremos con el nombre original por seguridad.")
					finalTargetFolder = similarFolder
				} else {
					fmt.Printf("-> ¡Carpeta renombrada exitosamente a '%s'!\n", finalTargetFolder)
				}
				break
			} else if resp == "3" {
				fmt.Printf("-> Crearemos una carpeta nueva: '%s'\n", finalTargetFolder)
				break
			} else {
				fmt.Println("Opción inválida.")
			}
		}
	}

	fmt.Println("\n----------------------------------------------")
	fmt.Println("Introduce la fecha de captura. Opciones:")
	ahora := time.Now()
	hoyStr := fmt.Sprintf("%d %s", ahora.Day(), mesesEs[int(ahora.Month())])
	esteAñoStr := fmt.Sprintf("%d", ahora.Year())

	fmt.Printf(" [ENTER vacío] -> Usa hoy: %s (Año: %s)\n", hoyStr, esteAñoStr)
	fmt.Printf(" '12 feb'      -> Usa esa fecha (Año: %s)\n", esteAñoStr)
	fmt.Println(" '12 feb 2025' -> Usa esa fecha y ese año")
	fmt.Print("Fecha: ")

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
			fmt.Println("🚨 ADVERTENCIA: SESIÓN EXISTENTE DETECTADA 🚨")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Printf("Ya existe una captura para '%s' en la fecha %s %s.\n", finalTargetFolder, finalDate, finalYear)
			fmt.Printf("Ruta: %s\n", targetPath)
			fmt.Println("Además, la carpeta YA CONTIENE ARCHIVOS adentro (fotos, logs, etc).")
			fmt.Println("Tomar el mismo objeto, 2 veces, en el mismo día exacto es poco común.")

			fmt.Print("\n¿Estás seguro de que deseas mezclar sesiones nuevas en esta fecha? (y/n) [n]: ")
			resp := strings.ToLower(readInput(reader))
			if resp != "y" {
				fmt.Println("Operación cancelada. (No se creó ni modificó ninguna carpeta).")
				fmt.Println("\nPresiona Enter para salir...")
				readInput(reader)
				return
			}
		}
	}

	for _, folder := range subfolders {
		folderPath := filepath.Join(targetPath, filepath.FromSlash(folder))
		err := os.MkdirAll(folderPath, 0755)
		if err != nil {
			fmt.Printf("❌ Error creando subcarpeta %s: %v\n", folder, err)
			return
		}
	}

	fmt.Println("\n✅ ¡Estructura generada exitosamente!")
	fmt.Printf("📁 Ruta creada: %s\n", targetPath)
	fmt.Printf("📂 Subcarpetas generadas: %s\n", strings.Join(subfolders, ", "))

	fmt.Print("\n¿Deseas MOVER tus fotos (Lights/Flats) hacia estas nuevas carpetas? (y/n) [n]: ")
	respMove := strings.ToLower(readInput(reader))
	if respMove == "y" {
		fmt.Print("\nArrastra aquí la CARPETA de tus Lights (o deja vacío para saltar): ")
		lightsSrc := cleanPath(readInput(reader))

		fmt.Print("Arrastra aquí la CARPETA de tus Flats (o deja vacío para saltar): ")
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
				fmt.Print("\n⚠️  ADVERTENCIA: Se detectaron posibles duplicados en el destino. Se renombrarán anexando el sufijo _1, _2... ¿Deseas continuar y duplicarlos? (y/n) [n]: ")
				respDup := strings.ToLower(readInput(reader))
				if respDup != "y" {
					fmt.Println("Operación de movimiento de archivos cancelada.")
					goto END_MOVE
				}
			}

			fmt.Println("\nPreparando archivos para mover...")
			var totalBytes int64
			var movedBytes int64

			if lightsSrc != "" {
				totalBytes += calculateTotalSize(lightsSrc)
			}
			if flatsSrc != "" {
				totalBytes += calculateTotalSize(flatsSrc)
			}

			fmt.Println("Comenzando transferencia...")
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

			fmt.Printf("\rProgreso: [==================================================] 100%% | ETA: 0s          \n")
			fmt.Println("\n¡Proceso de movimiento finalizado!")
		}
	}

END_MOVE:
	fmt.Println("\nPresiona Enter para salir...")
	readInput(reader)
}
