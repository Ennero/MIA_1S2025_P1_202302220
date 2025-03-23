package commands

import (
	structures "backend/structures"
	utils "backend/utils"
	"encoding/binary"
	"errors" // Paquete para manejar errores y crear nuevos errores con mensajes personalizados
	"fmt"    // Paquete para formatear cadenas y realizar operaciones de entrada/salida
	"os"
	"regexp"  // Paquete para trabajar con expresiones regulares, útil para encontrar y manipular patrones en cadenas
	"strconv" // Paquete para convertir cadenas a otros tipos de datos, como enteros
	"strings" // Paquete para manipular cadenas, como unir, dividir, y modificar contenido de cadenas
)

// FDISK estructura que representa el comando fdisk con sus parámetros
type FDISK struct {
	size int    // Tamaño de la partición
	unit string // Unidad de medida del tamaño (K o M)
	fit  string // Tipo de ajuste (BF, FF, WF)
	path string // Ruta del archivo del disco
	typ  string // Tipo de partición (P, E, L)
	name string // Nombre de la partición
}

/*
	fdisk -size=1 -type=L -unit=M -fit=BF -name="Particion3" -path="/home/keviin/University/PRACTICAS/MIA_LAB_S2_2024/CLASEEXTRA/disks/Disco1.mia"
	fdisk -size=300 -path=/home/Disco1.mia -name=Particion1
	fdisk -type=E -path=/home/Disco2.mia -Unit=K -name=Particion2 -size=300
*/

// CommandFdisk parsea el comando fdisk y devuelve una instancia de FDISK
func ParseFdisk(tokens []string) (string, error) {
	cmd := &FDISK{} // Crea una nueva instancia de FDISK

	// Unir tokens en una sola cadena y luego dividir por espacios, respetando las comillas
	args := strings.Join(tokens, " ")
	// Expresión regular para encontrar los parámetros del comando fdisk
	re := regexp.MustCompile(`-size=\d+|-unit=[kKmM]|-fit=[bBfF]{2}|-path="[^"]+"|-path=[^\s]+|-type=[pPeElL]|-name="[^"]+"|-name=[^\s]+`)
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
		case "-size":
			// Convierte el valor del tamaño a un entero
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				return "", errors.New("el tamaño debe ser un número entero positivo")
			}
			cmd.size = size
		case "-unit":
			// Verifica que la unidad sea "K" o "M"
			if value != "K" && value != "M" {
				return "", errors.New("la unidad debe ser K o M")
			}
			cmd.unit = strings.ToUpper(value)
		case "-fit":
			// Verifica que el ajuste sea "BF", "FF" o "WF"
			value = strings.ToUpper(value)
			if value != "BF" && value != "FF" && value != "WF" {
				return "", errors.New("el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = value
		case "-path":
			// Verifica que el path no esté vacío
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-type":
			// Verifica que el tipo sea "P", "E" o "L"
			value = strings.ToUpper(value)
			if value != "P" && value != "E" && value != "L" {
				return "", errors.New("el tipo debe ser P, E o L")
			}
			cmd.typ = value
		case "-name":
			// Verifica que el nombre no esté vacío
			if value == "" {
				return "", errors.New("el nombre no puede estar vacío")
			}
			cmd.name = value
		default:
			// Si el parámetro no es reconocido, devuelve un error
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Verifica que los parámetros -size, -path y -name hayan sido proporcionados
	if cmd.size == 0 {
		return "", errors.New("faltan parámetros requeridos: -size")
	}
	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}
	if cmd.name == "" {
		return "", errors.New("faltan parámetros requeridos: -name")
	}

	// Si no se proporcionó la unidad, se establece por defecto a "M"
	if cmd.unit == "" {
		cmd.unit = "M"
	}

	// Si no se proporcionó el ajuste, se establece por defecto a "FF"
	if cmd.fit == "" {
		cmd.fit = "FF"
	}

	// Si no se proporcionó el tipo, se establece por defecto a "P"
	if cmd.typ == "" {
		cmd.typ = "P"
	}

	// Crear la partición con los parámetros proporcionados
	err := commandFdisk(cmd)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	// Devuelve un mensaje de éxito con los detalles de la partición creada
	return fmt.Sprintf("FDISK: Partición creada exitosamente\n"+
		"-> Path: %s\n"+
		"-> Nombre: %s\n"+
		"-> Tamaño: %d%s\n"+
		"-> Tipo: %s\n"+
		"-> Fit: %s",
		cmd.path, cmd.name, cmd.size, cmd.unit, cmd.typ, cmd.fit), nil
}

func commandFdisk(fdisk *FDISK) error {
	// Convertir el tamaño a bytes
	sizeBytes, err := utils.ConvertToBytes(fdisk.size, fdisk.unit)
	if err != nil {
		fmt.Println("Error convirtiendo tamaño:", err)
		return err
	}

	switch fdisk.typ {
	case "P":
		// Crear partición primaria
		err = createPrimaryPartition(fdisk, sizeBytes)
		if err != nil {
			fmt.Println("Error creando partición primaria:", err)
			return err
		}
	case "E":
		// Crear partición extendida
		err = createExtendedPartition(fdisk, sizeBytes)
		if err != nil {
			fmt.Println("Error creando partición primaria:", err)
			return err
		}
	case "L":
		// Crear partición lógica
		err = createLogicalPartition(fdisk, sizeBytes)
		if err != nil {
			fmt.Println("Error creando partición primaria:", err)
			return err
		}
	}
	if err != nil {
		fmt.Println("Error creando partición:", err)
		return err
	}

	return nil
}


func createPrimaryPartition(fdisk *FDISK, sizeBytes int) error {
	// Crear una instancia de MBR
	var mbr structures.MBR

	// Deserializar la estructura MBR desde un archivo binario
	err := mbr.Deserialize(fdisk.path)
	if err != nil {
		fmt.Println("Error deserializando el MBR:", err)
		return err
	}

	/* SOLO PARA VERIFICACIÓN */
	// Imprimir MBR
	fmt.Println("\nMBR original:")
	mbr.PrintMBR()

	// Obtener la primera partición disponible
	availablePartition, startPartition, indexPartition := mbr.GetFirstAvailablePartition()
	if availablePartition == nil {
		fmt.Println("No hay particiones disponibles.")
	}

	for _, partitionName := range mbr.GetPartitionNames() {
		if partitionName == fdisk.name {
			fmt.Println("Ya existe una partición con el nombre especificado.")
			return errors.New("ya existe una partición con el nombre especificado")
		}
	}


	/* SOLO PARA VERIFICACIÓN */
	// Print para verificar que la partición esté disponible
	fmt.Println("\nPartición disponible:")
	availablePartition.PrintPartition()

	// Crear la partición con los parámetros proporcionados
	availablePartition.CreatePartition(startPartition, sizeBytes, fdisk.typ, fdisk.fit, fdisk.name)

	// Print para verificar que la partición se haya creado correctamente
	fmt.Println("\nPartición creada (modificada):")
	availablePartition.PrintPartition()

	// Colocar la partición en el MBR
	if availablePartition != nil {
		mbr.Mbr_partitions[indexPartition] = *availablePartition
	}

	// Imprimir las particiones del MBR
	fmt.Println("\nParticiones del MBR:")
	mbr.PrintPartitions()

	// Serializar el MBR en el archivo binario
	err = mbr.Serialize(fdisk.path)
	if err != nil {
		fmt.Println("Error:", err)
	}
	return nil
}


// Función para crear una partición extendida
func createExtendedPartition(fdisk *FDISK, sizeBytes int) error {
	var mbr structures.MBR

	// Deserializar el MBR del disco
	err := mbr.Deserialize(fdisk.path)
	if err != nil {
		fmt.Println("Error deserializando el MBR:", err)
		return err
	}

	// Verificar si ya existe una partición extendida
	for _, partition := range mbr.Mbr_partitions {
		if partition.Part_type[0] == 'E' {
			return errors.New("ya existe una partición extendida en el disco")
		}
	}

	// Obtener la primera partición disponible
	availablePartition, startPartition, indexPartition := mbr.GetFirstAvailablePartition()
	if availablePartition == nil {
		return errors.New("no hay espacio disponible para la partición extendida")
	}


	for _, partitionName := range mbr.GetPartitionNames() {
		if partitionName == fdisk.name {
			fmt.Println("Ya existe una partición con el nombre especificado.")
			return errors.New("ya existe una partición con el nombre especificado")
		}
	}


	// Crear la partición extendida
	availablePartition.CreatePartition(startPartition, sizeBytes, "E", fdisk.fit, fdisk.name)

	// Asignar la partición en el MBR
	mbr.Mbr_partitions[indexPartition] = *availablePartition

	// Serializar el MBR modificado
	err = mbr.Serialize(fdisk.path)
	if err != nil {
		fmt.Println("Error serializando MBR:", err)
		return err
	}

	fmt.Println("Partición extendida creada correctamente.")
	return nil
}



// Función para crear una partición lógica
func createLogicalPartition(fdisk *FDISK, sizeBytes int) error {
    var mbr structures.MBR

    // Deserializar el MBR
    err := mbr.Deserialize(fdisk.path)
    if err != nil {
        fmt.Println("Error deserializando MBR:", err)
        return err
    }

    // Buscar la partición extendida
    var extendedPartition *structures.Partition
    for i := range mbr.Mbr_partitions {
        if mbr.Mbr_partitions[i].Part_type[0] == 'E' {
            extendedPartition = &mbr.Mbr_partitions[i]
            break
        }
    }

    if extendedPartition == nil {
        return errors.New("no se encontró una partición extendida en el disco")
    }

    // Abrir el archivo del disco
    file, err := os.OpenFile(fdisk.path, os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    ebrSize := int32(binary.Size(structures.EBR{}))
    
    // Buscar el último EBR dentro de la partición extendida
    var lastEBR structures.EBR
    var currentEBRPosition int32 = extendedPartition.Part_start
    var lastEBRPosition int32 = -1
    var isFirstEBR bool = true

	//Comienzo con el ciclo para irme moviendo :)
    for {
        // Moverse al offset actual
        file.Seek(int64(currentEBRPosition), 0)
        
        // Leer el EBR en la posición actual
        err := binary.Read(file, binary.LittleEndian, &lastEBR)
        if err != nil {
            break
        }

        // Si es el primer EBR y está vacío (sin partición lógica)
        if isFirstEBR && lastEBR.Part_size <= 0 {
            break
        }
        
        isFirstEBR = false
        lastEBRPosition = currentEBRPosition

        // Si no hay más EBRs, salimos del bucle
        if lastEBR.Part_next == -1 {
            break
        }

        // Avanzar al siguiente EBR
        currentEBRPosition = lastEBR.Part_next
    }

    // Calcular la posición para el nuevo EBR
    var newEBRPosition int32
    var newPartitionStart int32
    
    if lastEBRPosition == -1 {
        // Si no hay EBRs previos, colocamos el nuevo EBR al inicio de la partición extendida
        newEBRPosition = extendedPartition.Part_start
        newPartitionStart = newEBRPosition + ebrSize  // La partición comienza después del EBR
    } else {
        // Si hay EBRs previos, calculamos la posición después del último EBR + su partición
        newEBRPosition = lastEBR.Part_start + lastEBR.Part_size
        newPartitionStart = newEBRPosition + ebrSize  // La partición comienza después del EBR
        
        // Verificar que haya espacio suficiente en la partición extendida
        if newPartitionStart + int32(sizeBytes) > extendedPartition.Part_start + extendedPartition.Part_size {
            return errors.New("no hay espacio suficiente en la partición extendida para la nueva partición lógica")
        }
    }

    // Crear el nuevo EBR para la partición lógica
    newEBR := structures.EBR{
        Part_status: [1]byte{'0'},
        Part_fit:    [1]byte{fdisk.fit[0]},
        Part_start:  newPartitionStart,  // La partición lógica comienza después del EBR
        Part_size:   int32(sizeBytes),
        Part_next:   -1,
    }

    // Copiar el nombre de la partición al EBR
    copy(newEBR.Part_name[:], fdisk.name)

    // Escribir el nuevo EBR en el archivo del disco
    file.Seek(int64(newEBRPosition), 0)
    err = binary.Write(file, binary.LittleEndian, &newEBR)
    if err != nil {
        fmt.Println("Error escribiendo EBR:", err)
        return err
    }

    // Actualizar el EBR anterior si existe
    if lastEBRPosition != -1 {
        lastEBR.Part_next = newEBRPosition
        file.Seek(int64(lastEBRPosition), 0)
        err = binary.Write(file, binary.LittleEndian, &lastEBR)
        if err != nil {
            fmt.Println("Error actualizando EBR anterior:", err)
            return err
        }
    }

    fmt.Println("Partición lógica creada correctamente.")
    return nil
}