package commands

import (
	stores "backend/stores"
	structures "backend/structures"
	"errors"
	"fmt"
	"regexp" // Paquete para trabajar con expresiones regulares, útil para encontrar y manipular patrones en cadenas
	"strings"
)

func ParseCat(tokens []string) (string, error) {
	// Verificar que se proporcionó un parámetro
	if len(tokens) == 0 {
		return "", fmt.Errorf("faltan parámetros requeridos")
	}

	// Unir tokens en una sola cadena y luego dividir por espacios, respetando las comillas
	args := strings.Join(tokens, " ")
	fmt.Printf("Argumentos completos: %s\n", args)

	//Expresión regular para encontrar todos los path
	re := regexp.MustCompile(`-file\d+="([^"]+)"`)

	// Buscar todas las coincidencias
	matches := re.FindAllStringSubmatch(args, -1)
	fmt.Printf("Coincidencias encontradas: %v\n", matches)

	// Extraer los paths en una lista
	var paths []string
	for _, match := range matches {
		if len(match) > 1 {
			path := match[1]
			// Asegurarse de que el path comience con /
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			paths = append(paths, path)
		}
	}

	// Si no se encontraron paths con el formato -fileN="path", intentar usar los tokens directamente
	if len(paths) == 0 {
		for _, token := range tokens {
			// Eliminar comillas si están presentes
			path := strings.Trim(token, "\"'")
			// Asegurarse de que el path comience con /
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			paths = append(paths, path)
		}
	}

	fmt.Println("---------------------------------")
	fmt.Println(paths)

	// Si aún no hay paths, reportar error
	if len(paths) == 0 {
		return "", fmt.Errorf("no se encontraron paths válidos en los argumentos")
	}

	texto, err := commandCat(paths)

	if err != nil {
		return "", err
	}

	// Devolver el contenido del archivo
	return fmt.Sprintf("CAT: Contenido de el/los archivo:\n%s", texto), nil
}

func commandCat(paths []string) (string, error) {
	var salidaBuilder strings.Builder

	var partitionID string
	if stores.Auth.IsAuthenticated() {
		partitionID = stores.Auth.GetPartitionID()
	} else {
		return "", errors.New("no se ha iniciado sesión en ninguna partición")
	}

	partitionSuperblock, _, partitionPath, err := stores.GetMountedPartitionSuperblock(partitionID) // Usar la función unificada
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada '%s': %w", partitionID, err)
	}

	// Validar superbloque por si acaso
	if partitionSuperblock.S_magic != 0xEF53 {
		return "", fmt.Errorf("magia del superbloque inválida (0x%X) para la partición '%s'", partitionSuperblock.S_magic, partitionID)
	}
	if partitionSuperblock.S_inode_size <= 0 || partitionSuperblock.S_block_size <= 0 {
		return "", fmt.Errorf("tamaño de inodo/bloque inválido en superbloque partición '%s'", partitionID)
	}

	for i, path := range paths {
		fmt.Printf("Procesando path [%d]: %s\n", i+1, path)

		if !strings.HasPrefix(path, "/") {
			return "", fmt.Errorf("error interno: path '%s' no es absoluto", path)
		}

		// Buscar Inodo
		inodeIndex, inode, errFind := structures.FindInodeByPath(partitionSuperblock, partitionPath, path)
		if errFind != nil {
			// Si no se encuentra, DEBE continuar con los otros archivos si hay más
			fmt.Printf("Error: No se encontró el archivo '%s': %v\n", path, errFind)
			salidaBuilder.WriteString(fmt.Sprintf("cat: %s: No such file or directory\n", path))
			continue
		}

		// Verificar si es archivo
		if inode.I_type[0] != '1' {
			fmt.Printf("Error: '%s' (inodo %d) no es un archivo (tipo: %c)\n", path, inodeIndex, inode.I_type[0])
			salidaBuilder.WriteString(fmt.Sprintf("cat: %s: Is a directory\n", path))
			continue
		}

		fmt.Printf("Archivo '%s' encontrado (inodo %d, tamaño %d bytes).\n", path, inodeIndex, inode.I_size)

		// Leer Contenido
		content, errRead := structures.ReadFileContent(partitionSuperblock, partitionPath, inode)
		if errRead != nil {
			fmt.Printf("Error leyendo contenido de '%s': %v\n", path, errRead)
			salidaBuilder.WriteString(fmt.Sprintf("cat: %s: Read error: %v\n", path, errRead))
			continue
		}

		if content == "" {
			fmt.Printf("Archivo '%s' está vacío.\n", path)

		}

		// Añadir contenido al resultado
		fmt.Printf("Contenido leído para '%s': %d bytes\n", path, len(content))
		salidaBuilder.WriteString(content)
	}

	return salidaBuilder.String(), nil // Retornar contenido acumulado
}
