package structures

import (
	utils "backend/utils"
	"fmt"
	"os"
	"strings"
	"time"
)


// Crear users.txt en nuestro sistema de archivos
func (sb *SuperBlock) CreateUsersFile(path string) error {

	// Validar tamaños para evitar división por cero más adelante
	if sb.S_inode_size <= 0 || sb.S_block_size <= 0 {
		return fmt.Errorf("tamaño de inodo o bloque inválido en superbloque: inode=%d, block=%d", sb.S_inode_size, sb.S_block_size)
	}

	// ----------- Se crea / -----------
	// Se calcula índices ANTES de modificar S_first_*
	rootInodeIndex := (sb.S_first_ino - sb.S_inode_start) / sb.S_inode_size
	rootBlockIndex := (sb.S_first_blo - sb.S_block_start) / sb.S_block_size

	// Crear el inodo raíz
	rootInode := &Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{rootBlockIndex, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'0'},
		I_perm:  [3]byte{'7', '7', '7'},
	}

	// Serializar el inodo raíz en la posición S_first_ino
	err := rootInode.Serialize(path, int64(sb.S_first_ino))
	if err != nil {
		return fmt.Errorf("error serializando inodo raíz: %w", err)
	}

	// Actualizar el bitmap de inodos en el índice calculado
	err = sb.UpdateBitmapInode(path, rootInodeIndex)
	if err != nil {
		return fmt.Errorf("error actualizando bitmap para inodo raíz (índice %d): %w", rootInodeIndex, err)
	}

	// Actualizar el superbloque (parte inodo)
	sb.S_free_inodes_count--
	sb.S_first_ino += sb.S_inode_size

	// Creamos el bloque del Inodo Raíz
	rootBlock := &FolderBlock{
		B_content: [4]FolderContent{
			{B_name: [12]byte{'.'}, B_inodo: rootInodeIndex},      // Apunta a sí mismo (índice 0)
			{B_name: [12]byte{'.', '.'}, B_inodo: rootInodeIndex}, // El padre de la raíz es la raíz (índice 0)
			{B_name: [12]byte{'-'}, B_inodo: -1},
			{B_name: [12]byte{'-'}, B_inodo: -1},
		},
	}

	// Serializar el bloque raíz en la posición S_first_blo
	err = rootBlock.Serialize(path, int64(sb.S_first_blo))
	if err != nil {
		return fmt.Errorf("error serializando bloque raíz: %w", err)
	}

	// Actualizar el bitmap de bloques en el índice calculado
	err = sb.UpdateBitmapBlock(path, rootBlockIndex)
	if err != nil {
		return fmt.Errorf("error actualizando bitmap para bloque raíz (índice %d): %w", rootBlockIndex, err)
	}

	// Actualizar el superbloque (parte bloque)
	sb.S_free_blocks_count--
	sb.S_first_blo += sb.S_block_size

	// ----------- Creamos /users.txt ---------------------------------------------------------------------------------------------------------------
	usersText := "1,G,root\n1,U,root,root,123\n"

	// Calcular índices para users.txt ANTES de modificar S_first_*
	usersInodeIndex := (sb.S_first_ino - sb.S_inode_start) / sb.S_inode_size
	usersBlockIndex := (sb.S_first_blo - sb.S_block_start) / sb.S_block_size

	// Actualizar la entrada en el bloque raíz para que apunte a users.txt
	if err := rootBlock.Deserialize(path, int64(sb.S_block_start)); err != nil { // Usa offset calculado o conocido
		return fmt.Errorf("error re-deserializando bloque raíz para actualizar: %w", err)
	}
	rootBlock.B_content[2] = FolderContent{B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't'}, B_inodo: usersInodeIndex} // Apunta al índice calculado
	if err := rootBlock.Serialize(path, int64(sb.S_block_start)); err != nil {
		return fmt.Errorf("error re-serializando bloque raíz actualizado: %w", err)
	}

	// Crear el inodo users.txt
	usersInode := &Inode{
		I_uid: 1, I_gid: 1,
		I_size:  int32(len(usersText)),
		I_atime: float32(time.Now().Unix()), I_ctime: float32(time.Now().Unix()), I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{usersBlockIndex, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // Usa índice calculado
		I_type:  [1]byte{'1'}, I_perm: [3]byte{'7', '7', '7'},
	}

	// Serializar inodo users.txt en S_first_ino
	err = usersInode.Serialize(path, int64(sb.S_first_ino))
	if err != nil {
		return fmt.Errorf("error serializando inodo users.txt: %w", err)
	}

	// Actualizar bitmap inodo en usersInodeIndex
	err = sb.UpdateBitmapInode(path, usersInodeIndex)
	if err != nil {
		return fmt.Errorf("error actualizando bitmap para inodo users.txt (índice %d): %w", usersInodeIndex, err)
	}

	// Actualizar el superbloque (parte inodo)
	sb.S_free_inodes_count--
	sb.S_first_ino += sb.S_inode_size

	// Crear el bloque de users.txt
	usersBlock := &FileBlock{B_content: [64]byte{}}
	copy(usersBlock.B_content[:], usersText)

	// Serializar el bloque de users.txt en S_first_blo
	err = usersBlock.Serialize(path, int64(sb.S_first_blo))
	if err != nil {
		return fmt.Errorf("error serializando bloque users.txt: %w", err)
	}

	// Actualizar bitmap bloque en usersBlockIndex
	err = sb.UpdateBitmapBlock(path, usersBlockIndex)
	if err != nil {
		return fmt.Errorf("error actualizando bitmap para bloque users.txt (índice %d): %w", usersBlockIndex, err)
	}

	// Actualizar el superbloque (parte bloque)
	sb.S_free_blocks_count--
	sb.S_first_blo += sb.S_block_size

	return nil
}

// CreateFolder crea una carpeta en el sistema de archivos
func (sb *SuperBlock) createFolderInInode(path string, inodeIndex int32, parentsDir []string, destDir string) error {
	// Validar tamaños para evitar división por cero más adelante
	if sb.S_inode_size <= 0 || sb.S_block_size <= 0 {
		return fmt.Errorf("tamaño de inodo o bloque inválido en superbloque: inode=%d, block=%d", sb.S_inode_size, sb.S_block_size)
	}

	// Deserializar inodo padre
	parentInode := &Inode{}
	parentInodeOffset := int64(sb.S_inode_start + (inodeIndex * sb.S_inode_size))
	err := parentInode.Deserialize(path, parentInodeOffset)
	if err != nil {
		return fmt.Errorf("error deserializando inodo padre %d: %w", inodeIndex, err)
	}

	// Verificar que el inodo padre sea un directorio
	if parentInode.I_type[0] != '0' {
		// Esto no debería pasar si la lógica de búsqueda es correcta, pero es una buena verificación
		return fmt.Errorf("intentando crear carpeta dentro de un archivo (inodo %d)", inodeIndex)
	}

	// Iterar sobre cada bloque del inodo padre
	for _, blockIndexInParent := range parentInode.I_block {
		if blockIndexInParent == -1 {
			continue // Puntero no usado
		}

		// Deserializar el bloque del directorio padre
		parentFolderBlock := &FolderBlock{}
		parentFolderBlockOffset := int64(sb.S_block_start + (blockIndexInParent * sb.S_block_size))
		err := parentFolderBlock.Deserialize(path, parentFolderBlockOffset)
		if err != nil {
			fmt.Printf("Advertencia: error deserializando bloque de directorio %d del padre %d: %v\n", blockIndexInParent, inodeIndex, err)
			continue // Intentar con el siguiente bloque del padre
		}

		// Si parentsDir no está vacío, buscar el subdirectorio intermedio
		if len(parentsDir) != 0 {
			targetSubDir := parentsDir[0]
			remainingPath := utils.RemoveElement(parentsDir, 0) // Path restante

			for _, content := range parentFolderBlock.B_content {
				if content.B_inodo != -1 {
					contentName := strings.TrimRight(string(content.B_name[:]), "\x00 ")
					if strings.EqualFold(contentName, targetSubDir) {
						// Encontrado el siguiente subdirectorio, llamar recursivamente
						err = sb.createFolderInInode(path, content.B_inodo, remainingPath, destDir)
						// Si la llamada recursiva tuvo éxito (o falló definitivamente), retornar
						return err
					}
				}
			}
			// Podría estar en otro bloque del padre, así que continuamos el bucle exterior
			continue
		}

		// Buscar un slot libre en este bloque del directorio padre
		foundSlot := false
		slotIndex := -1
		for i := 0; i < 4; i++ { // Asumiendo 4 entradas por FolderBlock
			if parentFolderBlock.B_content[i].B_inodo == -1 {
				slotIndex = i
				foundSlot = true
				break
			}
		}

		if foundSlot {
			//fmt.Printf("Encontrado slot libre %d en bloque %d del padre %d\n", slotIndex, blockIndexInParent, inodeIndex)

			// Calcular índices para la nueva carpeta y su bloque ANTES de actualizar SB
			newFolderInodeIndex := (sb.S_first_ino - sb.S_inode_start) / sb.S_inode_size
			newFolderBlockIndex := (sb.S_first_blo - sb.S_block_start) / sb.S_block_size

			// 1. Actualizar entrada en el bloque del directorio padre
			parentFolderBlock.B_content[slotIndex].B_inodo = newFolderInodeIndex
			copy(parentFolderBlock.B_content[slotIndex].B_name[:], destDir)
			// Serializar el bloque del directorio padre MODIFICADO
			err = parentFolderBlock.Serialize(path, parentFolderBlockOffset)
			if err != nil {
				return fmt.Errorf("error serializando bloque padre %d actualizado: %w", blockIndexInParent, err)
			}

			// Crear y serializar el Inodo de la nueva carpeta
			folderInode := &Inode{
				I_uid:   1, // TODO: Heredar o usar UID actual
				I_gid:   1, // TODO: Heredar o usar GID actual
				I_size:  0, // Tamaño inicial 0
				I_atime: float32(time.Now().Unix()),
				I_ctime: float32(time.Now().Unix()),
				I_mtime: float32(time.Now().Unix()),
				I_block: [15]int32{newFolderBlockIndex, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // Apunta al bloque calculado
				I_type:  [1]byte{'0'},                                                                           // Tipo Directorio
				I_perm:  [3]byte{'7', '7', '5'},                                                                 // Permisos (ej: rwxrwxr-x) TODO: Usar umask o permisos del padre?
			}
			err = folderInode.Serialize(path, int64(sb.S_first_ino))
			if err != nil {
				// Aquí podríamos intentar revertir el cambio en el bloque padre si la creación falla.
				return fmt.Errorf("error serializando inodo de nueva carpeta '%s': %w", destDir, err)
			}

			// Actualizar bitmap de inodos y Superbloque (parte inodo)
			err = sb.UpdateBitmapInode(path, newFolderInodeIndex)
			if err != nil {
				return fmt.Errorf("error actualizando bitmap para inodo %d ('%s'): %w", newFolderInodeIndex, destDir, err)
			}
			sb.S_free_inodes_count--
			sb.S_first_ino += sb.S_inode_size

			// Crear y serializar el Bloque de la nueva carpeta
			folderBlock := &FolderBlock{
				B_content: [4]FolderContent{
					{B_name: [12]byte{'.'}, B_inodo: newFolderInodeIndex}, // '.' apunta a sí mismo
					{B_name: [12]byte{'.', '.'}, B_inodo: inodeIndex},     // '..' apunta al padre
					{B_name: [12]byte{'-'}, B_inodo: -1},
					{B_name: [12]byte{'-'}, B_inodo: -1},
				},
			}
			err = folderBlock.Serialize(path, int64(sb.S_first_blo))
			if err != nil {
				// Revertir asignación de inodo sería complejo, mejor fallar.
				return fmt.Errorf("error serializando bloque para nueva carpeta '%s': %w", destDir, err)
			}

			// Actualizar bitmap de bloques y Superbloque (parte bloque)
			err = sb.UpdateBitmapBlock(path, newFolderBlockIndex)
			if err != nil {
				return fmt.Errorf("error actualizando bitmap para bloque %d ('%s'): %w", newFolderBlockIndex, destDir, err)
			}
			sb.S_free_blocks_count--
			sb.S_first_blo += sb.S_block_size

			return nil // Carpeta creada exitosamente en este bloque del padre
		}
	}

	// Si se recorrieron todos los bloques del padre y no se encontró espacio O no se encontró el subdirectorio intermedio
	if len(parentsDir) != 0 {
		return fmt.Errorf("no se encontró el subdirectorio intermedio '%s' en la ruta", parentsDir[0])
	} else {
		return fmt.Errorf("no se encontró espacio en los bloques existentes del directorio padre (inodo %d) para crear '%s'", inodeIndex, destDir)
	}
}

// COSITAS PARA LOS GRUPOS

// Actualiza el bitmap de bloques y el contador de bloques libres
func FreeInodeBlocks(inode *Inode, sb *SuperBlock, partitionPath string) error {
	fmt.Printf("Liberando bloques para inodo con tamaño %d...\n", inode.I_size)
	if inode.I_size == 0 { // Si el tamaño es 0
		// Podemos verificar I_block por si acaso, pero es probable que estén en -1
		fmt.Println("Tamaño de inodo es 0, no se liberan bloques.")

		for i := range inode.I_block {
			inode.I_block[i] = -1
		}
		return nil
	}

	// Liberar bloques directos
	for i := 0; i < 12; i++ {
		if err := freeDataBlockIfValid(inode.I_block[i], sb, partitionPath); err != nil {
			fmt.Printf("Error liberando bloque directo %d: %v\n", inode.I_block[i], err)
		}
		inode.I_block[i] = -1 // Marcar como libre
	}
	// Liberar bloques simples
	if err := freeIndirectBlocksRecursive(1, inode.I_block[12], sb, partitionPath); err != nil {
		fmt.Printf("Error liberando indirección simple (nivel 1 desde %d): %v\n", inode.I_block[12], err)
	}
	inode.I_block[12] = -1

	// Liberar bloques dobles
	if err := freeIndirectBlocksRecursive(2, inode.I_block[13], sb, partitionPath); err != nil {
		fmt.Printf("Error liberando indirección doble (nivel 2 desde %d): %v\n", inode.I_block[13], err)
	}
	inode.I_block[13] = -1

	// Liberar bloques triples
	if err := freeIndirectBlocksRecursive(3, inode.I_block[14], sb, partitionPath); err != nil {
		fmt.Printf("Error liberando indirección triple (nivel 3 desde %d): %v\n", inode.I_block[14], err)
	}
	inode.I_block[14] = -1

	return nil
}

// Libera los bloques de datos/punteros inferiores y LUEGO el bloque de punteros actual
func freeIndirectBlocksRecursive(level int, blockPtr int32, sb *SuperBlock, partitionPath string) error {
	if level < 1 || level > 3 || blockPtr == -1 || blockPtr >= sb.S_blocks_count {
		return nil
	}

	// Deserializar el bloque de punteros de este nivel
	ptrBlock := &PointerBlock{}
	ptrOffset := int64(sb.S_block_start) + int64(blockPtr)*int64(sb.S_block_size)
	if err := ptrBlock.Deserialize(partitionPath, ptrOffset); err != nil {
		fmt.Printf("Advertencia: no se pudo leer bloque de punteros Nivel %d (%d): %v. Intentando liberar bloque %d de todas formas.\n", level, blockPtr, err, blockPtr)
		return freeDataBlockIfValid(blockPtr, sb, partitionPath) // Intentar liberar el ptrBlock mismo
	}

	for _, nextPtr := range ptrBlock.P_pointers {
		var errRec error
		if level == 1 { 
			errRec = freeDataBlockIfValid(nextPtr, sb, partitionPath)
		} else { // Niveles superiores, nextPtr apunta a OTRO bloque de punteros
			errRec = freeIndirectBlocksRecursive(level-1, nextPtr, sb, partitionPath)
		}
		if errRec != nil {
			// Loguear pero continuar para intentar liberar el resto
			fmt.Printf("Error durante liberación recursiva (nivel %d->%d, desde %d a %d): %v\n", level, level-1, blockPtr, nextPtr, errRec)
		}
	}

	// Después de liberar/procesar todos los punteros internos, liberar el bloque de punteros actual
	fmt.Printf("Liberando bloque de punteros Nivel %d (índice %d)\n", level, blockPtr)
	return freeDataBlockIfValid(blockPtr, sb, partitionPath)
}

func freeDataBlockIfValid(blockIndex int32, sb *SuperBlock, partitionPath string) error {
	if blockIndex == -1 || blockIndex < 0 || blockIndex >= sb.S_blocks_count {
		return nil // Índice inválido o no usado, nada que hacer
	}

	// Actualizar bitmap
	bitmapOffset := int64(sb.S_bm_block_start) + int64(blockIndex)
	file, err := os.OpenFile(partitionPath, os.O_WRONLY, 0644) // Solo escritura
	if err != nil {
		return fmt.Errorf("error abriendo disco para liberar bloque %d: %w", blockIndex, err)
	}
	defer file.Close()

	_, err = file.Seek(bitmapOffset, 0)
	if err != nil {
		return fmt.Errorf("error buscando bitmap para liberar bloque %d: %w", blockIndex, err)
	}

	_, err = file.Write([]byte{'0'}) // Marcar como libre
	if err != nil {
		return fmt.Errorf("error escribiendo en bitmap para liberar bloque %d: %w", blockIndex, err)
	}

	// Actualizar contador de libres en Superbloque (EN MEMORIA)
	sb.S_free_blocks_count++

	fmt.Printf("Bloque %d marcado como libre.\n", blockIndex)
	return nil
}
