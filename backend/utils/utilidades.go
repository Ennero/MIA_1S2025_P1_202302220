package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//LO TIPICO BÁSICO --------------------------------------------------------------------------------------------------------------------------------
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
var cuentaPathALetra = make(map[string]int)

//Indice de la siguiente letra disponible
var siguienteLetra=0

//Función para obtener la letra de cada path
func AsignarLetra(path string) (string, int, error) {
	//Comienzo asignando una letra si, es que no la tiene asignada
	if _,ok:=pathALetra[path];!ok{ //Si no tiene asignada una letra
		if siguienteLetra<len(letras){ //Si hay letras disponibles
			pathALetra[path]=letras[siguienteLetra] //Asigno la letra
			cuentaPathALetra[path]=0 //Inicializo la cuenta de la letra
			siguienteLetra++ 
		}else{ //Si no hay más letras disponibles
			fmt.Println("ERROR: No hay más letras disponibles")
			return "",0,errors.New("No hay más letras disponibles")
		}
	}
	cuentaPathALetra[path]++ //Aumento la cuenta de la letra
	siguienteIndice := cuentaPathALetra[path] //Obtengo el índice de la letra
	return pathALetra[path],siguienteIndice,nil //Devuelvo la letra y el índice
}


//PARTE PARA LOS DIRECTORIOS --------------------------------------------------------------------------------------------------------------------------

//Función para crear un directorio padre
func CrearDirectorioPadre(path string) error{
	dir := filepath.Dir(path) //Obtengo el directorio padre

	err := os.MkdirAll(dir, os.ModePerm) //Creo el directorio padre

	if err != nil { //Si hay un error, devuelvo el error
		return fmt.Errorf("Error al crear el directorio padre: %s", err)
	}
	return nil
}

// Función que obtiene el nombre del archivo .dot y el nombre de la imagen de salida
func ObtenerNombreArchivos(path string) (string, string) {
	dir := filepath.Dir(path)//Obtengo el directorio del archivo
	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))//Obtengo el nombre del archivo sin la extensión
	dotFileName := filepath.Join(dir, baseName+".dot")//Creo el nombre del archivo .dot
	outputImage := path//Creo el nombre de la imagen de salida
	return dotFileName, outputImage//Devuelvo los nombres
}

// Función para obtener las carpetas padres y el directorio de destino
func ObtenerDiretoriosPadres(path string) ([]string, string) {
	path = filepath.Clean(path) // Limpiar el path

	components := strings.Split(path, string(filepath.Separator))//Dividir el path en sus componentes

	var parentDirs []string // Arreglo para almacenar las carpetas padres

	//Ciclo para obtener las carpetas padres (todas menos la última)
	for i := 1; i < len(components)-1; i++ {
		parentDirs = append(parentDirs, components[i])
	}

	destDir := components[len(components)-1] // Obtener la última carpeta (directorio de destino)

	return parentDirs, destDir //Retorna las carpetas padres y el directorio de destino
}

//PARTE PARA MANEJO PARTICIONES --------------------------------------------------------------------------------------------------------------------------
//Función para obtener el primer elemento de un slice
func PrimerElementoSlice[T any](slice []T) (T, error) {

	//Si el slice está vacío, devuelvo un error
	if len(slice) == 0 {
		var zero T
		return zero, errors.New("el slice está vacío")
	}
	return slice[0], nil
}

//Función para eliminar un elemento de un slice
func EliminarElementoSlice[T any](slice []T, index int) []T {
	if index < 0 || index >= len(slice) {
		return slice // Índice fuera de rango, devolver el slice original
	}
	return append(slice[:index], slice[index+1:]...)
}

//Función que divide una cadena en partes de tamaño chunksize y las almacena en una lista
func DividirCadenaEnChunk(s string) []string{
	var chunk []string //Arreglo para almacenar los chunks

	//Ciclo para dividir la cadena en chunks de 64 caracteres
	for i := 0; i < len(s); i += 64 {
		end := i + 64
		if end > len(s) {
			end = len(s)
		}
		//Agregar el chunk a la lista
		chunk = append(chunk, s[i:end])
	}
	return chunk //Retorna la lista de chunks
}
