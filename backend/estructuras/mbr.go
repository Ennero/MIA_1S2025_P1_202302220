package estructuras

import (
	"bytes" // Importa el paquete "bytes" para manipular secuencias de bytes
	"encoding/binary" // Importa el paquete "binary" para codificar y decodificar datos en y desde representaciones binarias
	"fmt" // Importa el paquete "fmt" para formatear e imprimir texto
	"os" // Importa el paquete "os" para manejar archivos y directorios
	"strings" // Importa el paquete "strings" para manipulación de cadenas
	"time" // Importa el paquete "time" para manejar fechas y horas
)


type MBR struct {
	mbr_tamano          int32
	mbr_fecha_creacion     float32
	Mbr_disk_signature     int16
	dsk_fit 		  [1]byte
	mbr_partitions    [4]Particion
}

func (mbr *MBR) SerializarMBR (path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	//Aqui serializo la estructura MBR en el archivo
	err = binary.Write(file, binary.LittleEndian, mbr)
	if err != nil {
		return	err
	}
	return nil
}


// Deserializa el MBR y lee la estructura MBR desde el inicio de un archivo binario
func (mbr *MBR) DeserializarMBR(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	//Obtengo el tamaño de la estructura MBR
	mbrSize := binary.Size(mbr)
	if mbrSize <= 0 {
		return fmt.Errorf("Tamaño inválido: %d", mbrSize)
	}
	//Leo solo la cantidad de bytes que corresponden al tamaño de la estructura MBR
	buffer := make([]byte, mbrSize)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	//Deserializo los bytes leídos en la estructura MBR
	reader := bytes.NewReader(buffer)
	err = binary.Read(reader, binary.LittleEndian, mbr)
	if err != nil {
		return err
	}
	return nil
}

// Método para obtener la primera partición disponible
func (mbr *MBR) ObtenerPrimerParticion() (*Particion, int, int) {
	// Calcular el offset para el comienzo de la partición
	offset := binary.Size(mbr) // Tamaño del MBR en bytes

	// Recorro las particiones del MBR
	for i := 0; i < len(mbr.mbr_partitions); i++ {

		// Si el start de la partición es -1, entonces está disponible
		if mbr.mbr_partitions[i].part_start == -1 {
			return &mbr.mbr_partitions[i], offset, i //Retorno la partición, el offset y el índice
		} else {
			// Calculo el nuevo offset para la siguiente partición, es decir, sumar el tamaño de la partición
			offset += int(mbr.mbr_partitions[i].part_s)
		}
	}
	return nil, -1, -1 // Si no hay particiones disponibles, entonces retorno nil
}

// Método para obtener una partición por nombre
func (mbr *MBR) ObtenerParcionPorNombre(name string) (*Particion, int) {

	// Recorro las particiones del MBR
	for i, partition := range mbr.mbr_partitions {
		
		//Convertir part_name a string y elimino los caracteres nulos
		partitionName := strings.Trim(string(partition.part_name[:]), "\x00 ")

		//Converto el nombre de la partición a string y elimino los caracteres nulos
		inputName := strings.Trim(name, "\x00 ")

		//Si el nombre de la partición coincide, devolver la partición y el índice
		if strings.EqualFold(partitionName, inputName) {
			return &partition, i
		}
	}
	return nil, -1
}

// Método para imprimir los valores del MBR
func (mbr *MBR) PrintMBR() {
	// Convertir Mbr_creation_date a time.Time
	creationTime := time.Unix(int64(mbr.mbr_fecha_creacion), 0)

	// Convertir Mbr_disk_fit a char
	diskFit := rune(mbr.dsk_fit[0])

	fmt.Printf("MBR Tamano: %d\n", mbr.mbr_tamano)
	fmt.Printf("Fecha de creación: %s\n", creationTime.Format(time.RFC3339))
	fmt.Printf("Firma de id del disco: %d\n", mbr.Mbr_disk_signature)
	fmt.Printf("Particiones: %c\n", diskFit)
}

// Método para imprimir las particiones del MBR para ir debugeando
func (mbr *MBR) PrintPartitions() {

	//Recorro todas la parciones del MBR
	for i, partition := range mbr.mbr_partitions {
		// Convertir Part_status, Part_type y Part_fit a char
		partStatus := rune(partition.part_status[0])
		partType := rune(partition.part_type[0])
		partFit := rune(partition.part_fit[0])

		// Convertir Part_name a string
		partName := string(partition.part_name[:])
		// Convertir Part_id a string
		partID := string(partition.part_id[:])

		fmt.Printf("Particion %d:\n", i+1)
		fmt.Printf("  Status: %c\n", partStatus)
		fmt.Printf("  Tipo: %c\n", partType)
		fmt.Printf("  Fit: %c\n", partFit)
		fmt.Printf("  Inicio: %d\n", partition.part_start)
		fmt.Printf("  Tamano: %d\n", partition.part_s)
		fmt.Printf("  Nombre: %s\n", partName)
		fmt.Printf("  Correlativo: %d\n", partition.part_correlative)
		fmt.Printf("  ID: %s\n", partID)
	}
}





