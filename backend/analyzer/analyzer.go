package analyzer

import (
	commands "backend/commands"
	"errors"  // Importa el paquete "errors" para manejar errores
	"fmt"     // Importa el paquete "fmt" para formatear e imprimir texto
	"strings" // Importa el paquete "strings" para manipulación de cadenas
)

// Analyzer analiza el comando de entrada y ejecuta la acción correspondiente
func Analyzer(input string) (string, error) {
	// Divide la entrada en tokens usando espacios en blanco como delimitadores
	tokens := strings.Fields(input)

	// Si no se proporcionó ningún comando, devuelve un error
	if len(tokens) == 0 {
		return "", errors.New("no se proporcionó ningún comando")
	}

	//Transforma el primer token a minúsculas
	tokens[0] = strings.ToLower(tokens[0])

	// Switch para manejar diferentes comandos
	switch tokens[0] {
	case "mkdisk":
		// Llama a la función ParseMkdisk del paquete commands con los argumentos restantes
		return commands.ParseMkdisk(tokens[1:])
	case "fdisk":
		// Llama a la función CommandFdisk del paquete commands con los argumentos restantes
		return commands.ParseFdisk(tokens[1:])
	case "mount":
		// Llama a la función CommandMount del paquete commands con los argumentos restantes
		return commands.ParseMount(tokens[1:])
	case "mkfs":
		// Llama a la función CommandMkfs del paquete commands con los argumentos restantes
		return commands.ParseMkfs(tokens[1:])
	case "rep":
		// Llama a la función CommandRep del paquete commands con los argumentos restantes
		return commands.ParseRep(tokens[1:])
	case "mkdir":
		// Llama a la función CommandMkdir del paquete commands con los argumentos restantes
		return commands.ParseMkdir(tokens[1:])
	case "rmdisk":
		// Llama a la función CommandRmdisk del paquete commands con los argumentos restantes
		return commands.ParseRmdisk(tokens[1:])
	case "mounted":
		// Llama la función CommandMounted del paquete commands con los argumentos restantes
		return commands.ParseMounted(tokens[1:])
	case "cat":
		// Llama la función CommandCat del paquete commands con los argumentos restantes
		return commands.ParseCat(tokens[1:])
	case "login":
		// Llama a la función ParseLogin del paquete commands con los argumentos restantes
		return commands.ParseLogin(tokens[1:])
	case "logout":
		// Llama a la función ParseLogout del paquete commands con los argumentos restantes
		return commands.ParseLogout(tokens[1:])
	case "mkfile":
		// Llama a la función ParseMkfile del paquete commands con los argumentos restantes
		return commands.ParseMkfile(tokens[1:])
	case "mkgrp":
		// Llama a la función ParseMkgrp del paquete commands con los argumentos restantes
		return commands.ParseMkgrp(tokens[1:])
	default:
		if tokens[0][0] == '#' {
			// Si el primer carácter del comando es '#', se considera un comentario
			return "", nil
		}
		// Si el comando no es reconocido, devuelve un error
		return "", fmt.Errorf("comando desconocido: %s", tokens[0])
	}
}
