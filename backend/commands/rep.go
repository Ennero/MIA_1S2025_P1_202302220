package commands

import (
	reports "backend/reports"
	stores "backend/stores"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// REP estructura que representa el comando rep con sus parámetros
type REP struct {
	id           string // ID del disco
	path         string // Ruta del archivo del disco
	name         string // Nombre del reporte
	path_file_ls string // Ruta del archivo ls (opcional)
}

// ParserRep parsea el comando rep y devuelve una instancia de REP
func ParseRep(tokens []string) (string, error) {
	cmd := &REP{} // Crea una nueva instancia de REP

	// Unir tokens en una sola cadena y luego dividir por espacios, respetando las comillas
	args := strings.Join(tokens, " ")
	// Expresión regular para encontrar los parámetros del comando rep
	re := regexp.MustCompile(`-id=[^\s]+|-path="[^"]+"|-path=[^\s]+|-name=[^\s]+|-path_file_ls="[^"]+"|-path_file_ls=[^\s]+`)
	// Encuentra todas las coincidencias de la expresión regular en la cadena de argumentos
	matches := re.FindAllString(args, -1)

	// Itera sobre cada coincidencia encontrada
	for _, match := range matches {
		// Divide cada parte en clave y valor usando "=" como delimitador
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]

		// Remove quotes from value if present
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		// Switch para manejar diferentes parámetros
		switch key {
		case "-id":
			// Verifica que el id no esté vacío
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		case "-path":
			// Verifica que el path no esté vacío
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-name":
			// Verifica que el nombre sea uno de los valores permitidos
			validNames := []string{"mbr", "disk", "inode", "block", "bm_inode", "bm_block", "sb", "file", "ls", "tree"}
			if !contains(validNames, value) {
				return "", errors.New("nombre inválido, debe ser uno de los siguientes: mbr, disk, inode, block, bm_inode, bm_block, sb, file, ls, tree")
			}
			cmd.name = value
		case "-path_file_ls":
			cmd.path_file_ls = value
		default:
			// Si el parámetro no es reconocido, devuelve un error
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Verifica que los parámetros obligatorios hayan sido proporcionados
	if cmd.id == "" || cmd.path == "" || cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -id, -path, -name")
	}

	if cmd.name == "file" && cmd.path_file_ls == "" {
		return "", errors.New("el parámetro -path_file_ls es requerido para el reporte 'file'")
	}

	if cmd.name == "ls" && cmd.path_file_ls == "" {
		return "", errors.New("el parámetro -path_file_ls es requerido para el reporte 'ls'")
	}

	// Aquí se puede agregar la lógica para ejecutar el comando rep con los parámetros proporcionados
	err := commandRep(cmd)
	if err != nil {
		fmt.Println("Error:", err)
	}

	return fmt.Sprintf("REP: Reporte generado exitosamente\n"+
		"-> ID: %s\n"+
		"-> Path: %s\n"+
		"-> Tipo: %s%s",
		cmd.id,
		cmd.path,
		cmd.name,
		func() string {
			if cmd.path_file_ls != "" {
				return fmt.Sprintf("\n-> Path LS: %s", cmd.path_file_ls)
			}
			return ""
		}()), nil
}

// Función auxiliar para verificar si un valor está en una lista
func contains(list []string, value string) bool {
	return slices.Contains(list, value)
}

func commandRep(rep *REP) error {
	// Obtener la partición montada
	mountedMbr, mountedSb, mountedDiskPath, err := stores.GetMountedPartitionRep(rep.id)
	if err != nil {
		return err
	}

	// Switch para manejar diferentes tipos de reportes
	switch rep.name {
	case "mbr":
		err = reports.ReportMBR(mountedMbr, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
	case "inode":
		err = reports.ReportInode(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err

		}
	case "bm_inode":
		err = reports.ReportBMInode(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err

		}
	case "disk":
		err = reports.ReportDisk(mountedMbr, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err

		}
	case "bm_block":
		err = reports.ReportBMBlock(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err

		}
	case "sb":
		err = reports.ReportSuperBlock(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err

		}
	case "block":
		err = reports.ReportBlock(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
	case "tree":
		err = reports.ReportTree(mountedSb, mountedDiskPath, rep.path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return	 err
		}

	case "file":
		err = reports.ReportFile(mountedSb, mountedDiskPath, rep.path, rep.path_file_ls)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
	case "ls":
		err = reports.ReportLS(mountedSb, mountedDiskPath, rep.path, rep.path_file_ls)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}

	}

	return nil
}
