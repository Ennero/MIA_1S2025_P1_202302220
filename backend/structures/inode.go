package structures

import (
	"bytes"
	"encoding/binary"
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
func FindInodeByPath(sb *SuperBlock, diskPath string, path string) (*Inode, error) {

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
			return nil, fmt.Errorf("error al leer inodo raíz: %v", err)
		}
		return rootInode, nil
	}

	currentInodeNum := int32(0) // Inodo raíz es 0

	// Para cada componente del path, buscar en el directorio correspondiente
	for i, component := range cleanComponents {

		fmt.Printf("Buscando componente %d: %s (en inodo %d)\n", i, component, currentInodeNum)

		currentInode := &Inode{}
		offset := int64(sb.S_inode_start + currentInodeNum*sb.S_inode_size)
		if err := currentInode.Deserialize(diskPath, offset); err != nil {
			return nil, err
		}

		fmt.Printf("Tipo de inodo actual: %s\n", string(currentInode.I_type[:]))

		// Verificar que el inodo actual es un directorio (excepto para el último componente)
		if i < len(cleanComponents)-1 && currentInode.I_type[0] != '0' {
			return nil, fmt.Errorf("'%s' no es un directorio", component)
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
				return nil, fmt.Errorf("error al leer bloque %d: %v", blockPtr, err)
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
			return nil, fmt.Errorf("no se encontró '%s' en el directorio actual", component)
		}
	}

	// Leer el inodo final
	targetInode := &Inode{}
	offset := int64(sb.S_inode_start + currentInodeNum*sb.S_inode_size)
	fmt.Printf("Obteniendo inodo final %d en offset %d\n", currentInodeNum, offset)

	// Debugeando
	if err := targetInode.Deserialize(diskPath, offset); err != nil {
		return nil, fmt.Errorf("error al leer inodo final %d: %v", currentInodeNum, err)
	}

	// Debugeando
	fmt.Printf("Inodo encontrado - tipo: %s, tamaño: %d\n",
		string(targetInode.I_type[:]), targetInode.I_size)

	return targetInode, nil
}

func ReadFileContent(sb *SuperBlock, diskPath string, inode *Inode) (string, error) {
	content := make([]byte, 0, inode.I_size)

	//Debugeando
	fmt.Printf("Leyendo archivo: tamaño esperado %d bytes\n", inode.I_size)
	fmt.Printf("Bloques directos: %v\n", inode.I_block[:12])

	readBlock := func(blockPtr int32) error {
		if blockPtr == -1 {
			return nil
		}

		fmt.Printf("Leyendo bloque %d en posición %d\n",
			blockPtr, sb.S_block_start+blockPtr*sb.S_block_size)

		fileBlock := &FileBlock{}
		blockOffset := int64(sb.S_block_start + blockPtr*sb.S_block_size)
		if err := fileBlock.Deserialize(diskPath, blockOffset); err != nil {
			return err
		}

		// Print the first few bytes of the block for debugging
		preview := string(fileBlock.B_content[:min(20, len(fileBlock.B_content))])
		fmt.Printf("Vista previa del bloque %d: %q...\n", blockPtr, preview)

		remaining := inode.I_size - int32(len(content))
		if remaining <= 0 {
			return nil
		}
		toCopy := min(int(remaining), len(fileBlock.B_content))
		content = append(content, fileBlock.B_content[:toCopy]...)

		fmt.Printf("Contenido actual (después de leer bloque %d): %d bytes\n",
			blockPtr, len(content))

		return nil
	}

	// Verificar si el tamaño del inodo es válido
	if inode.I_size <= 0 {
		return "", fmt.Errorf("tamaño de archivo inválido: %d", inode.I_size)
	}

	// Bloques directos (0-11)
	for i := 0; i < 12; i++ {

		if inode.I_block[i] == -1 {
			continue
		}

		if err := readBlock(inode.I_block[i]); err != nil {
			return "", err
		}
		if int32(len(content)) >= inode.I_size {
			break
		}
	}

	// Bloque indirecto simple (12)
	if inode.I_block[12] != -1 {

		fmt.Printf("Leyendo bloque indirecto %d\n", inode.I_block[12])

		ptrBlock := &PointerBlock{}
		ptrOffset := int64(sb.S_block_start + inode.I_block[12]*sb.S_block_size)
		if err := ptrBlock.Deserialize(diskPath, ptrOffset); err != nil {
			return "", fmt.Errorf("error al leer bloque indirecto: %v", err)
		}
		for _, ptr := range ptrBlock.P_pointers {
			if ptr == -1 {
				continue
			}
			if err := readBlock(ptr); err != nil {
				return "", err
			}
			if int32(len(content)) >= inode.I_size {
				break
			}
		}
	}

	// Asegurarse de que no excedemos el tamaño declarado del archivo
	if int32(len(content)) > inode.I_size {
		content = content[:inode.I_size]
	}

	result := string(content)
	fmt.Printf("Contenido final: %d bytes, string resultado: %q\n", len(result), result)

	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
