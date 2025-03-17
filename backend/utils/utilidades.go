package backend

import (
	"errors" // Importa el paquete "errors" para manejar errores
	"fmt"    // Importa el paquete "fmt" para formatear e imprimir texto
)

//Función para pasar todo a bytes porque que pereza hacerlo todo a cada rato
func PasarABytes(tam int, unidad string) (int, error) {
	switch unidad {
	case "b":
		return tam, nil //Si la unidad es en bytes, devuelvo el mismo tamaño
	case "k":
		return tam * 1024, nil //Multiplico por 1024 para obtener los bytes
	case "m":
		return tam * 1024 * 1024, nil //Multiplico por 1024 dos veces para obtener los bytes
		default:
		return 0, errors.New("Unidad no reconocida") //Devuelvo un error si la unidad no es válida
}}

//Arreglo con todas las letras del abecedario para asignarlas después a la partición
var letras = []string{"A","B","C","D","E","F","G","H","I","J","K","L","M","N","O","P","Q","R","S","T","U","V","W","X","Y","Z"}

//Mapita para asignar letras a los paths
var pathALetra=make(map[string]string)

//Indice de la siguiente letra disponible
var siguienteLetra=0

//Función para obtener la letra de cada path
func AsignarLetra(path string) (string,error) {
	//Comienzo asignando una letra si es que no la tiene asignada
	if _,ok:=pathALetra[path];!ok{
		if siguienteLetra<len(letras){
			pathALetra[path]=letras[siguienteLetra]
			siguienteLetra++
		}else{
			fmt.Println("ERROR: No hay más letras disponibles")
			return "",errors.New("No hay más letras disponibles")
		}
	}
	return pathALetra[path],nil
}
