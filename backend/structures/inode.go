package structures

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Inode struct {
	I_uid   int32
	I_gid   int32
	I_size  int32
	I_atime float32
	I_ctime float32
	I_mtime float32
	I_block [15]int32
	I_type  [1]byte
	I_perm  [3]byte
	// Total: 88 bytes
}

// Serialize escribe la estructura Inode en un archivo binario en la posición especificada
func (inode *Inode) Serialize(path string, offset int64) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Mover el puntero del archivo a la posición especificada
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	// Serializar la estructura Inode directamente en el archivo
	err = binary.Write(file, binary.LittleEndian, inode)
	if err != nil {
		return err
	}

	return nil
}

// Deserialize lee la estructura Inode desde un archivo binario en la posición especificada
func (inode *Inode) Deserialize(path string, offset int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Mover el puntero del archivo a la posición especificada
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	// Obtener el tamaño de la estructura Inode
	inodeSize := binary.Size(inode)
	if inodeSize <= 0 {
		return fmt.Errorf("invalid Inode size: %d", inodeSize)
	}

	// Leer solo la cantidad de bytes que corresponden al tamaño de la estructura Inode
	buffer := make([]byte, inodeSize)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	// Deserializar los bytes leídos en la estructura Inode
	reader := bytes.NewReader(buffer)
	err = binary.Read(reader, binary.LittleEndian, inode)
	if err != nil {
		return err
	}

	return nil
}

// Print imprime los atributos del inodo
func (inode *Inode) Print() {
	atime := time.Unix(int64(inode.I_atime), 0)
	ctime := time.Unix(int64(inode.I_ctime), 0)
	mtime := time.Unix(int64(inode.I_mtime), 0)

	fmt.Printf("I_uid: %d\n", inode.I_uid)
	fmt.Printf("I_gid: %d\n", inode.I_gid)
	fmt.Printf("I_size: %d\n", inode.I_size)
	fmt.Printf("I_atime: %s\n", atime.Format(time.RFC3339))
	fmt.Printf("I_ctime: %s\n", ctime.Format(time.RFC3339))
	fmt.Printf("I_mtime: %s\n", mtime.Format(time.RFC3339))
	fmt.Printf("I_block: %v\n", inode.I_block)
	fmt.Printf("I_type: %s\n", string(inode.I_type[:]))
	fmt.Printf("I_perm: %s\n", string(inode.I_perm[:]))
}

// FUNCIÓN PARA BUSCAR UN ARCHIVO---------------------------------------------------------------------------------------
// FUNCIÓN PARA BUSCAR UN ARCHIVO---------------------------------------------------------------------------------------
func FindInodeByPath(sb *SuperBlock, diskPath string, path string) (int32, *Inode, error) {
	fmt.Printf("Buscando inodo para path: %s\n", path)

	components := strings.Split(path, "/")
	var cleanComponents []string
	for _, c := range components {
		if c != "" {
			cleanComponents = append(cleanComponents, c)
		}
	}

	fmt.Printf("Componentes del path: %v\n", cleanComponents)

	// Si es un path vacío o solo la raíz (/), devolver el inodo raíz
	if len(cleanComponents) == 0 {
		rootInode := &Inode{}
		if err := rootInode.Deserialize(diskPath, int64(sb.S_inode_start)); err != nil {
			return -1, nil, fmt.Errorf("error al leer inodo raíz: %v", err)
		}
		return 0, rootInode, nil
	}

	currentInodeNum := int32(0) // Inodo raíz es 0

	// Para cada componente del path, buscar en el directorio correspondiente
	for i, component := range cleanComponents {
		fmt.Printf("Buscando componente %d: %s (en inodo %d)\n", i, component, currentInodeNum)

		currentInode := &Inode{}
		offset := int64(sb.S_inode_start + currentInodeNum*sb.S_inode_size)
		if err := currentInode.Deserialize(diskPath, offset); err != nil {
			return -1, nil, err
		}

		fmt.Printf("Tipo de inodo actual: %s\n", string(currentInode.I_type[:]))

		// Verificar que el inodo actual es un directorio (excepto para el último componente)
		if i < len(cleanComponents)-1 && currentInode.I_type[0] != '0' {
			return -1, nil, fmt.Errorf("'%s' no es un directorio", component)
		}

		found := false
		// Iterar sobre los bloques de punteros del inodo actual
		for blockIndex, blockPtr := range currentInode.I_block {
			if blockPtr == -1 {
				continue
			}
			// Debugeando
			fmt.Printf("Examinando bloque %d (puntero %d) del inodo %d\n",
				blockIndex, blockPtr, currentInodeNum)

			// Leer el bloque de carpeta
			folderBlock := &FolderBlock{}
			blockOffset := int64(sb.S_block_start + blockPtr*sb.S_block_size)
			if err := folderBlock.Deserialize(diskPath, blockOffset); err != nil {
				return -1, nil, fmt.Errorf("error al leer bloque %d: %v", blockPtr, err)
			}

			// Imprimir el contenido del bloque para depuración
			fmt.Println("Contenido del bloque:")
			for entryIndex, entry := range folderBlock.B_content {
				if entry.B_inodo != -1 {
					name := strings.TrimRight(string(entry.B_name[:]), "\x00")
					fmt.Printf("  [%d] %q -> inodo %d\n", entryIndex, name, entry.B_inodo)
				}
			}

			// Buscar el componente actual en el bloque de carpeta
			for _, entry := range folderBlock.B_content {
				if entry.B_inodo == -1 {
					continue
				}

				// Convertir el nombre del archivo a una cadena y eliminar los caracteres nulos
				name := strings.TrimRight(string(entry.B_name[:]), "\x00")
				fmt.Printf("Comparando '%s' con '%s'\n", name, component)

				// Si el nombre del archivo coincide con el componente actual, actualizar el inodo actual
				if name == component {
					currentInodeNum = entry.B_inodo
					found = true
					fmt.Printf("¡Encontrado! El inodo para '%s' es %d\n", component, currentInodeNum)
					break
				}
			}
			// Si se encontró el componente actual, salir del bucle de bloques
			if found {
				break
			}
		}

		// Si no se encontró el componente actual, devolver un error
		if !found {
			return -1, nil, fmt.Errorf("no se encontró '%s' en el directorio actual", component)
		}
	}

	// Leer el inodo final
	targetInode := &Inode{}
	offset := int64(sb.S_inode_start + currentInodeNum*sb.S_inode_size)
	fmt.Printf("Obteniendo inodo final %d en offset %d\n", currentInodeNum, offset)

	// Debugeando
	if err := targetInode.Deserialize(diskPath, offset); err != nil {
		return -1, nil, fmt.Errorf("error al leer inodo final %d: %v", currentInodeNum, err)
	}

	// Debugeando
	fmt.Printf("Inodo encontrado - tipo: %s, tamaño: %d\n",
		string(targetInode.I_type[:]), targetInode.I_size)

	return currentInodeNum, targetInode, nil
}


// ReadFileContent lee el contenido completo de un archivo, manejando indirección.
func ReadFileContent(sb *SuperBlock, diskPath string, inode *Inode) (string, error) {
	if inode.I_type[0] != '1' {
		return "", fmt.Errorf("inodo %d no es un archivo", -1)
	}
	if inode.I_size <= 0 {
		return "", nil // Archivo vacío
	}
	if sb.S_block_size <= 0 {
		return "", errors.New("tamaño de bloque inválido en superbloque")
	}

	//Buffer para construir el contenido eficientemente
	var content bytes.Buffer
	content.Grow(int(inode.I_size)) // Pre-asignar capacidad

	// Función auxiliar para leer un bloque de datos y añadirlo al buffer
	readBlock := func(blockPtr int32) error {
		// Detener si ya se leyó suficiente
		if int32(content.Len()) >= inode.I_size {
			return nil
		}

		if blockPtr == -1 {
			return nil
		}
		if blockPtr < 0 || blockPtr >= sb.S_blocks_count {
			return fmt.Errorf("puntero de bloque de datos inválido: %d", blockPtr)
		}

		fileBlock := &FileBlock{}
		blockOffset := int64(sb.S_block_start) + int64(blockPtr)*int64(sb.S_block_size)
		if err := fileBlock.Deserialize(diskPath, blockOffset); err != nil {
			return fmt.Errorf("error leyendo bloque de datos %d: %w", blockPtr, err)
		}

		// Calcular cuántos bytes copiar de este bloque
		remainingInFile := inode.I_size - int32(content.Len())
		bytesToCopy := int(sb.S_block_size)
		if int32(bytesToCopy) > remainingInFile {
			bytesToCopy = int(remainingInFile)
		}

		if bytesToCopy > 0 {
			content.Write(fileBlock.B_content[:bytesToCopy])
		}
		return nil
	}

	// Bloques Directos (0-11)
	fmt.Println("Leyendo bloques directos...")
	for i := 0; i < 12; i++ {
		if err := readBlock(inode.I_block[i]); err != nil {
			return "", err
		}
		if int32(content.Len()) >= inode.I_size {
			break
		}
	}
	if int32(content.Len()) >= inode.I_size {
		return content.String(), nil
	}

	//Indirecto Simple (12)
	if inode.I_block[12] != -1 {
		fmt.Println("Leyendo bloques desde Indirecto Simple...")
		err := readIndirectBlocksRecursive(1, inode.I_block[12], sb, diskPath, &content, inode.I_size, readBlock)
		if err != nil {
			return "", fmt.Errorf("error en indirección simple: %w", err)
		}
		if int32(content.Len()) >= inode.I_size {
			return content.String(), nil
		}
	}

	// Indirecto Doble (13)
	if inode.I_block[13] != -1 {
		fmt.Println("Leyendo bloques desde Indirecto Doble...")
		err := readIndirectBlocksRecursive(2, inode.I_block[13], sb, diskPath, &content, inode.I_size, readBlock)
		if err != nil {
			return "", fmt.Errorf("error en indirección doble: %w", err)
		}
		if int32(content.Len()) >= inode.I_size {
			return content.String(), nil
		}
	}

	// Indirecto Triple (14)
	if inode.I_block[14] != -1 {
		fmt.Println("Leyendo bloques desde Indirecto Triple...")
		err := readIndirectBlocksRecursive(3, inode.I_block[14], sb, diskPath, &content, inode.I_size, readBlock)
		if err != nil {
			return "", fmt.Errorf("error en indirección triple: %w", err)
		}
	}

	// Asegurarse de no devolver más bytes que inode.I_size (aunque la lógica anterior debería prevenirlo)
	finalContent := content.Bytes()
	if int32(len(finalContent)) > inode.I_size {
		finalContent = finalContent[:inode.I_size]
	}

	return string(finalContent), nil
}


func readIndirectBlocksRecursive(
	level int, // Nivel de indirección actual (1, 2, 3)
	blockPtr int32, // Puntero al bloque de punteros de este nivel
	sb *SuperBlock,
	diskPath string,
	content *bytes.Buffer, // Usar buffer para eficiencia
	sizeLimit int32,
	readBlockFunc func(int32) error, // Función para leer un bloque de DATOS
) error {

	// Condición de parada: nivel inválido o puntero inválido
	if level < 1 || level > 3 || blockPtr == -1 || blockPtr >= sb.S_blocks_count {
		return nil // No es un error, simplemente no hay nada que leer aquí
	}
	// Detener si ya hemos leído suficiente
	if int32(content.Len()) >= sizeLimit {
		return nil
	}

	// Deserializar el bloque de punteros de este nivel
	ptrBlock := &PointerBlock{}
	ptrOffset := int64(sb.S_block_start) + int64(blockPtr)*int64(sb.S_block_size)
	if err := ptrBlock.Deserialize(diskPath, ptrOffset); err != nil {
		// Loguear error pero intentar continuar si es posible? O retornar error?
		fmt.Printf("Advertencia: error al leer bloque de punteros nivel %d (índice %d): %v\n", level, blockPtr, err)
		return nil // Podría ser un error fatal, pero intentamos ser robustos
	}

	// Iterar sobre los punteros de este bloque
	for _, nextPtr := range ptrBlock.P_pointers {
		if nextPtr == -1 {
			continue
		}
		if nextPtr < 0 || nextPtr >= sb.S_blocks_count {
			fmt.Printf("Advertencia: puntero inválido %d encontrado en bloque de punteros nivel %d (índice %d)\n", nextPtr, level, blockPtr)
			continue
		}

		// Si es el último nivel de indirección (nivel 1), los punteros apuntan a bloques de DATOS
		if level == 1 {
			if err := readBlockFunc(nextPtr); err != nil {
				return fmt.Errorf("error leyendo bloque de datos %d desde indirecto: %w", nextPtr, err)
			}
		} else {
			// Si no es el último nivel, llamar recursivamente para el siguiente nivel inferior
			err := readIndirectBlocksRecursive(level-1, nextPtr, sb, diskPath, content, sizeLimit, readBlockFunc)
			if err != nil {
				return err // Propagar error de niveles inferiores
			}
		}

		// Detener si ya hemos leído suficiente después de procesar un puntero
		if int32(content.Len()) >= sizeLimit {
			break
		}
	}
	return nil
}
