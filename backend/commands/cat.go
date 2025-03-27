package commands

import (
	stores "backend/stores"
	"fmt"
	"regexp" // Paquete para trabajar con expresiones regulares, útil para encontrar y manipular patrones en cadenas
	"strings"
	structures "backend/structures"
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

	// Aquí se puede agregar la lógica para ejecutar el comando mkdir con los parámetros proporcionados
	if err != nil {
		return "", err
	}

	// Devolver el contenido del archivo
	return fmt.Sprintf("CAT: Contenido de el/los archivo:\n%s", texto), nil
}



func commandCat(paths []string) (string, error) {
    salida := ""
    _, mountedSb, mountedDiskPath, err := stores.GetMountedPartitionRep(IdPartition)
    if err != nil {
        return "", err
    }

    for _, path := range paths {
		fmt.Println("Buscando path",path)


		if !strings.HasPrefix(path, "/") {
            path = "/" + path
        }

        _,inode, err := structures.FindInodeByPath(mountedSb, mountedDiskPath, path)
        if err != nil {
            return "", fmt.Errorf("error al buscar inodo: %v", err)
        }

		//Por si no se encontró el Inodo con el path
		if inode==nil {
			return "", fmt.Errorf("no se encontró el archivo")
		}


        if inode.I_type[0] != '1' {
            return "", fmt.Errorf("'%s' no es un archivo", path)
        }

        content, err := structures.ReadFileContent(mountedSb, mountedDiskPath, inode)
        if err != nil {
            return "", fmt.Errorf("error al leer contenido: %v", err)
        }

		//Debugenado por si está vacio :)
		if content=="" {
			return "", fmt.Errorf("el archivo está vacío")
		}

        salida += fmt.Sprintf("%s\n", content) // Revisar después si me dan ganas que se vea bonito
    }


    return salida, nil
}