package analizador

import (
	"errors"
	"fmt"
	"strings"
)

// La función que realiza el análisis y ejecuta los comandos
func Analizador(input string) (interface{}, error) {

	//Comienzo dividiendo los tokens del análisis léxico
	tokens := strings.Fields(input)

	//Si no hay tokens, devuelvo un error
	if len(tokens) == 0 {
		return nil, errors.New("No se ingresó ningún comando")
	}

	//Paso todo a minúsculas para evitar problemas de case sensitive
	for i := 0; i < len(tokens); i++ {
		tokens[i] = strings.ToLower(tokens[i])
	}

	//Aquí hago el switch case con todos los comandos posibles en el programa
	switch tokens[0] {
	case "mkdisk":
		return nil, nil

	case "fdisk":
		return nil, nil

	case "mount":
		return nil, nil

	case "mkfs":
		return nil, nil

	case "rep":
		return nil, nil

	case "mkdir":
		return nil, nil

	default:
		return nil, fmt.Errorf("Comando no reconocido: %s", tokens[0])

	}
}
