package commands

import (
	"fmt"
	"os" // Necesario para leer archivo con -cont
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	stores "backend/stores"
	structures "backend/structures"
	utils "backend/utils"
	"errors"
)

type MKFILE struct {
	path string // Path del archivo
	r    bool   // Crear padres recursivamente
	size int    // Tamaño en bytes (si no se usa -cont)
	cont string // Path al archivo local con contenido
}

// ParseMkfile analiza los tokens para el comando mkfile
func ParseMkfile(tokens []string) (string, error) {
	cmd := &MKFILE{size: 0} // Inicializar tamaño a 0 por defecto

	args := strings.Join(tokens, " ")
	// Expresión regular mejorada para capturar valores con/sin comillas y flags
	re := regexp.MustCompile(`-(path|cont)=("[^"]+"|[^\s]+)|-size=(\d+)|(-r)`)
	matches := re.FindAllStringSubmatch(args, -1) // Usar Submatch para capturar grupos

	parsedArgs := make(map[string]bool) // Para rastrear qué parte del string original ya se procesó
	for _, match := range matches {
		fullMatch := match[0]
		parsedArgs[fullMatch] = true     // Marcar el token completo como procesado
		key := strings.ToLower(match[1]) // path o cont (grupo 1)
		value := match[2]                // Valor para path/cont (grupo 2)
		sizeStr := match[3]              // Valor para size (grupo 3)
		flagR := match[4]                // -r (grupo 4)

		// Limpiar comillas del valor si existen
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch {
		case key == "path":
			cmd.path = value
		case key == "cont":
			cmd.cont = value
		case sizeStr != "":
			size, err := strconv.Atoi(sizeStr)
			if err != nil {
				return "", fmt.Errorf("valor de -size inválido: %s", sizeStr)
			}
			if size < 0 {
				return "", errors.New("el valor de -size no puede ser negativo")
			}
			cmd.size = size
		case flagR == "-r":
			cmd.r = true
		}
	}

	// Verificar si hubo tokens no reconocidos
	originalTokens := strings.Fields(args)
	for _, token := range originalTokens {
		isProcessed := false
		for parsed := range parsedArgs {
			if strings.Contains(parsed, token) || strings.Contains(token, parsed) {
				isProcessed = true
				break
			}
		}
		if token == "-r" {
			isProcessed = true
		}

		if !isProcessed && !strings.HasPrefix(token, "-") {
		} else if !isProcessed {
			return "", fmt.Errorf("parámetro o formato inválido cerca de: %s", token)
		}
	}
	// Validaciones obligatorias
	if cmd.path == "" {
		return "", errors.New("parámetro obligatorio faltante: -path")
	}
	if cmd.cont != "" && cmd.size != 0 && len(matches) > 0 {
		fmt.Println("Parámetro -size ignorado porque -cont fue proporcionado.")
		cmd.size = 0
	}
	// Validar existencia de archivo en -cont si se proporcionó
	if cmd.cont != "" {
		if _, err := os.Stat(cmd.cont); os.IsNotExist(err) {
			return "", fmt.Errorf("el archivo especificado en -cont no existe: %s", cmd.cont)
		}
	}
	err := commandMkfile(cmd)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MKFILE: Archivo '%s' creado correctamente.", cmd.path), nil
}

// commandMkfile contiene la lógica principal para crear el archivo
func commandMkfile(mkfile *MKFILE) error {
	//Obtener Autenticación y Partición Montada
	var userID int32 = 1 
	var groupID int32 = 1 
	var partitionID string

	if stores.Auth.IsAuthenticated() {
		partitionID = stores.Auth.GetPartitionID()
		fmt.Printf("Usuario autenticado: %s (Usando UID=1, GID=1 por defecto)\n", stores.Auth.Username)
	} else {
		return errors.New("no se ha iniciado sesión en ninguna partición")
	}

	partitionSuperblock, mountedPartition, partitionPath, err := stores.GetMountedPartitionSuperblock(partitionID)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada '%s': %w", partitionID, err)
	}

	// Validar tamaños para división por cero
	if partitionSuperblock.S_inode_size <= 0 || partitionSuperblock.S_block_size <= 0 {
		return fmt.Errorf("tamaño de inodo o bloque inválido en superbloque: inode=%d, block=%d", partitionSuperblock.S_inode_size, partitionSuperblock.S_block_size)
	}

	// Limpiar Path y Obtener Padre/Nombre
	cleanPath := strings.TrimSuffix(mkfile.path, "/")
	if !strings.HasPrefix(cleanPath, "/") {
		return errors.New("el path debe ser absoluto (empezar con /)")
	}
	if cleanPath == "/" {
		return errors.New("no se puede crear archivo en la raíz '/' con este comando")
	}
	if cleanPath == "" {
		return errors.New("el path no puede estar vacío")
	}

	parentPath := filepath.Dir(cleanPath)
	fileName := filepath.Base(cleanPath)
	if fileName == "" || fileName == "." || fileName == ".." {
		return fmt.Errorf("nombre de archivo inválido: %s", fileName)
	}
	if len(fileName) > 12 {
		return fmt.Errorf("el nombre del archivo '%s' excede los 12 caracteres permitidos", fileName)
	}

	// Asegurar que el nombre no contenga caracteres inválidos
	fmt.Printf("Asegurando directorio padre: %s\n", parentPath)
	parentInodeIndex, parentInode, err := ensureParentDirExists(parentPath, mkfile.r, partitionSuperblock, partitionPath)
	if err != nil {
		return err 
	}

	fmt.Printf("Verificando si '%s' ya existe en inodo %d...\n", fileName, parentInodeIndex)
	exists, _, existingInodeType := findEntryInParent(parentInode, fileName, partitionSuperblock, partitionPath)
	if exists {
		existingTypeStr := "elemento"
		if existingInodeType == '0' {
			existingTypeStr = "directorio"
		}
		if existingInodeType == '1' {
			existingTypeStr = "archivo"
		}
		return fmt.Errorf("error: el %s '%s' ya existe en '%s'", existingTypeStr, fileName, parentPath)
	}

	// Determinar Contenido y Tamaño
	var contentBytes []byte
	var fileSize int32

	if mkfile.cont != "" {
		fmt.Printf("Leyendo contenido desde archivo local: %s\n", mkfile.cont)
		hostContent, errRead := os.ReadFile(mkfile.cont)
		if errRead != nil {
			return fmt.Errorf("error leyendo archivo de contenido '%s': %w", mkfile.cont, errRead)
		}
		contentBytes = hostContent
		fileSize = int32(len(contentBytes))
	} else {
		fileSize = int32(mkfile.size)
		if fileSize > 0 {
			fmt.Printf("Generando contenido de %d bytes (0-9 repetido)...\n", fileSize)
			contentBuilder := strings.Builder{}
			for i := int32(0); i < fileSize; i++ {
				contentBuilder.WriteByte(byte('0' + (i % 10)))
			}
			contentBytes = []byte(contentBuilder.String())
		} else {
			contentBytes = []byte{}
		}
	}
	fmt.Printf("Tamaño final del archivo: %d bytes\n", fileSize)

	// Calcular bloques necesarios 
	blockSize := partitionSuperblock.S_block_size
	numBlocksNeeded := int32(0)
	if fileSize > 0 {
		numBlocksNeeded = (fileSize + blockSize - 1) / blockSize
	}

	// Asignar Bloques de Datos y Punteros
	fmt.Printf("Asignando %d bloque(s) de datos y punteros necesarios...\n", numBlocksNeeded)
	var allocatedBlockIndices [15]int32
	allocatedBlockIndices, err = allocateDataBlocks(contentBytes, fileSize, partitionSuperblock, partitionPath)
	if err != nil {
		return fmt.Errorf("falló la asignación de bloques: %w", err)
	}

	// Asignar Inodo
	fmt.Println("Asignando inodo...")
	newInodeIndex := (partitionSuperblock.S_first_ino - partitionSuperblock.S_inode_start) / partitionSuperblock.S_inode_size
	err = partitionSuperblock.UpdateBitmapInode(partitionPath, newInodeIndex)
	if err != nil {
		return fmt.Errorf("error actualizando bitmap para inodo %d: %w", newInodeIndex, err)
	}
	partitionSuperblock.S_free_inodes_count--
	partitionSuperblock.S_first_ino += partitionSuperblock.S_inode_size

	// Crear y Serializar Estructura Inodo
	currentTime := float32(time.Now().Unix())
	newInode := &structures.Inode{
		I_uid: userID, I_gid: groupID, I_size: fileSize,
		I_atime: currentTime, I_ctime: currentTime, I_mtime: currentTime,
		I_type: [1]byte{'1'}, I_perm: [3]byte{'6', '6', '4'},
	}
	newInode.I_block = allocatedBlockIndices

	inodeOffset := int64(partitionSuperblock.S_inode_start) + int64(newInodeIndex)*int64(partitionSuperblock.S_inode_size)
	err = newInode.Serialize(partitionPath, inodeOffset)
	if err != nil {
		return fmt.Errorf("error serializando nuevo inodo %d: %w", newInodeIndex, err)
	}

	// Añadir Entrada al Directorio Padre
	fmt.Printf("Añadiendo entrada '%s' al directorio padre (inodo %d)...\n", fileName, parentInodeIndex)
	err = addEntryToParent(parentInodeIndex, fileName, newInodeIndex, partitionSuperblock, partitionPath)
	if err != nil {
		return fmt.Errorf("error añadiendo entrada '%s' al directorio padre: %w", fileName, err)
	}

	// Serializar Superbloque
	fmt.Println("\nSerializando SuperBlock después de MKFILE...")
	err = partitionSuperblock.Serialize(partitionPath, int64(mountedPartition.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar el superbloque después de mkfile: %w", err)
	}
	return nil
}

// Retorna el índice y el inodo del padre directo si todo va bien.
func ensureParentDirExists(targetParentPath string, createRecursively bool, sb *structures.SuperBlock, partitionPath string) (int32, *structures.Inode, error) {
	fmt.Printf("Asegurando que exista: %s (Recursivo: %v)\n", targetParentPath, createRecursively)
	//El padre es la raíz "/"
	if targetParentPath == "/" {
		inode := &structures.Inode{}
		offset := int64(sb.S_inode_start) // Raíz es inodo 0
		err := inode.Deserialize(partitionPath, offset)
		if err != nil {
			return -1, nil, fmt.Errorf("error crítico: no se pudo deserializar inodo raíz (0): %w", err)
		}
		if inode.I_type[0] != '0' {
			return -1, nil, errors.New("error crítico: inodo raíz (0) no es un directorio")
		}
		return 0, inode, nil 
	}

	// Verificar si el padre objetivo ya existe
	parentInodeIndex, parentInode, errFind := structures.FindInodeByPath(sb, partitionPath, targetParentPath)

	if errFind == nil { // Padre encontrado
		// Verificar si es un directorio
		if parentInode.I_type[0] != '0' {
			return -1, nil, fmt.Errorf("error: el path padre '%s' existe pero no es un directorio", targetParentPath)
		}
		// Padre existe y es directorio, todo bien
		fmt.Printf("Directorio padre '%s' (inodo %d) encontrado.\n", targetParentPath, parentInodeIndex)
		return parentInodeIndex, parentInode, nil
	}

	// Padre no encontrado
	fmt.Printf("Padre '%s' no encontrado (%v).\n", targetParentPath, errFind)
	if !createRecursively {
		// Si no es recursivo, fallamos
		return -1, nil, fmt.Errorf("el directorio padre '%s' no existe y la opción -r no fue especificada", targetParentPath)
	}

	//Intentar crear el padre
	grandParentPath := filepath.Dir(targetParentPath)
	parentDirName := filepath.Base(targetParentPath)

	_, _, errEnsureGrandParent := ensureParentDirExists(grandParentPath, true, sb, partitionPath) // Llamada recursiva
	if errEnsureGrandParent != nil {
		// Si falla crear el abuelo, no podemos crear el padre
		return -1, nil, fmt.Errorf("error asegurando ancestro '%s': %w", grandParentPath, errEnsureGrandParent)
	}

	// Ahora que el abuelo, creamos el padre
	fmt.Printf("Creando directorio padre faltante: '%s' dentro de '%s'\n", parentDirName, grandParentPath)
	parentDirsForCreate, destDirForCreate := utils.GetParentDirectories(targetParentPath)
	errCreate := sb.CreateFolder(partitionPath, parentDirsForCreate, destDirForCreate)
	if errCreate != nil {
		return -1, nil, fmt.Errorf("falló la creación recursiva del directorio padre '%s': %w", targetParentPath, errCreate)
	}

	// Si llegamos aquí, buscamos de nuevo el padre recién creado
	fmt.Printf("Verificando padre recién creado '%s'\n", targetParentPath)
	parentInodeIndex, parentInode, errFindAgain := structures.FindInodeByPath(sb, partitionPath, targetParentPath)
	if errFindAgain != nil {
		return -1, nil, fmt.Errorf("error crítico: no se encontró el directorio padre '%s' después de crearlo: %w", targetParentPath, errFindAgain)
	}
	if parentInode.I_type[0] != '0' {
		return -1, nil, fmt.Errorf("error crítico: el directorio padre '%s' recién creado no es un directorio", targetParentPath)
	}

	fmt.Printf("Directorio padre '%s' (inodo %d) creado y verificado.\n", targetParentPath, parentInodeIndex)
	return parentInodeIndex, parentInode, nil
}

// Retorna si existe, el índice del inodo encontrado y su tipo
func findEntryInParent(parentInode *structures.Inode, entryName string, sb *structures.SuperBlock, partitionPath string) (exists bool, foundInodeIndex int32, foundInodeType byte) {
	exists = false
	foundInodeIndex = -1
	foundInodeType = '?'

	if parentInode.I_type[0] != '0' {
		return
	}

	for _, blockPtr := range parentInode.I_block {
		if blockPtr == -1 {
			continue
		}
		if blockPtr < 0 || blockPtr >= sb.S_blocks_count {
			continue
		}

		folderBlock := &structures.FolderBlock{}
		offset := int64(sb.S_block_start) + int64(blockPtr)*int64(sb.S_block_size)
		if err := folderBlock.Deserialize(partitionPath, offset); err != nil {
			fmt.Printf("Advertencia: No se pudo leer el bloque de directorio %d al buscar '%s'\n", blockPtr, entryName)
			continue
		}

		for _, content := range folderBlock.B_content {
			if content.B_inodo != -1 {
				name := strings.TrimRight(string(content.B_name[:]), "\x00")
				if name == entryName {
					exists = true
					foundInodeIndex = content.B_inodo
					tempInode := &structures.Inode{}
					tempOffset := int64(sb.S_inode_start) + int64(foundInodeIndex)*int64(sb.S_inode_size)
					if err := tempInode.Deserialize(partitionPath, tempOffset); err == nil {
						foundInodeType = tempInode.I_type[0]
					}
					return
				}
			}
		}
	}
	return
}

func addEntryToParent(parentInodeIndex int32, entryName string, entryInodeIndex int32, sb *structures.SuperBlock, partitionPath string) error {

	parentInode := &structures.Inode{}
	parentOffset := int64(sb.S_inode_start) + int64(parentInodeIndex)*int64(sb.S_inode_size)
	if err := parentInode.Deserialize(partitionPath, parentOffset); err != nil {
		return fmt.Errorf("no se pudo leer inodo padre %d para añadir entrada: %w", parentInodeIndex, err)
	}
	if parentInode.I_type[0] != '0' {
		return fmt.Errorf("el inodo padre %d no es un directorio", parentInodeIndex)
	}

	// Buscar slot libre en bloques existentes
	fmt.Printf("Buscando slot libre en bloques existentes del padre %d...\n", parentInodeIndex)
	// Función auxiliar para buscar en un bloque carpeta
	findAndAddInFolderBlock := func(blockPtr int32) (bool, error) {
		if blockPtr == -1 {
			return false, nil
		} // No es un bloque válido
		if blockPtr < 0 || blockPtr >= sb.S_blocks_count {
			fmt.Printf("Advertencia: Puntero inválido %d encontrado al buscar slot libre.\n", blockPtr)
			return false, nil // Saltar puntero inválido
		}

		folderBlock := &structures.FolderBlock{}
		blockOffset := int64(sb.S_block_start) + int64(blockPtr)*int64(sb.S_block_size)
		if err := folderBlock.Deserialize(partitionPath, blockOffset); err != nil {
			fmt.Printf("Advertencia: No se pudo leer bloque %d del padre %d para añadir entrada\n", blockPtr, parentInodeIndex)
			return false, nil
		}

		for i := 0; i < len(folderBlock.B_content); i++ {
			nameBytes := folderBlock.B_content[i].B_name[:]
			isDot := (nameBytes[0] == '.' && (len(nameBytes) < 2 || nameBytes[1] == 0))
			isDotDot := (nameBytes[0] == '.' && nameBytes[1] == '.' && (len(nameBytes) < 3 || nameBytes[2] == 0))

			if folderBlock.B_content[i].B_inodo == -1 && !isDot && !isDotDot { // Slot libre encontrado!
				fmt.Printf("Encontrado slot libre %d en bloque existente %d del padre %d\n", i, blockPtr, parentInodeIndex)
				folderBlock.B_content[i].B_inodo = entryInodeIndex
				copy(folderBlock.B_content[i].B_name[:], entryName)
				if err := folderBlock.Serialize(partitionPath, blockOffset); err != nil { // Serializar bloque modificado
					return false, fmt.Errorf("falló al escribir la nueva entrada en el bloque existente %d: %w", blockPtr, err)
				}
				// Actualizar tiempos del padre y serializar padre
				currentTime := float32(time.Now().Unix())
				parentInode.I_mtime = currentTime
				parentInode.I_atime = currentTime
				if err := parentInode.Serialize(partitionPath, parentOffset); err != nil {
					return false, fmt.Errorf("falló al actualizar tiempos del inodo padre %d tras añadir en bloque existente: %w", parentInodeIndex, err)
				}
				return true, nil
			}
		}
		return false, nil
	}

	// Buscar en bloques directos
	for k := 0; k < 12; k++ {
		found, err := findAndAddInFolderBlock(parentInode.I_block[k])
		if err != nil {
			return err
		} 
		if found {
			return nil
		} 
	}

	// Buscar en bloques apuntados por indirecto simple
	if parentInode.I_block[12] != -1 {
		fmt.Printf("Buscando slot libre en bloques de indirección simple (L1 en %d)...\n", parentInode.I_block[12])
		l1Block := &structures.PointerBlock{}
		l1Offset := int64(sb.S_block_start) + int64(parentInode.I_block[12])*int64(sb.S_block_size)
		if err := l1Block.Deserialize(partitionPath, l1Offset); err == nil {
			for _, folderBlockPtr := range l1Block.P_pointers {
				found, err := findAndAddInFolderBlock(folderBlockPtr)
				if err != nil {
					return err
				}
				if found {
					return nil
				}
			}
		} else {
			fmt.Printf("Advertencia: No se pudo leer el bloque de punteros L1 %d\n", parentInode.I_block[12])
		}
	}

	//Si no se encontró slot, buscar un PUNTERO libre para un NUEVO bloque
	fmt.Printf("No se encontró slot libre en bloques existentes del padre %d. Buscando puntero libre...\n", parentInodeIndex)

	// Función auxiliar para asignar y preparar un nuevo bloque carpeta
	allocateAndPrepareNewFolderBlock := func() (int32, *structures.FolderBlock, error) {
		if sb.S_free_blocks_count < 1 {
			return -1, nil, errors.New("no hay bloques libres para expandir directorio")
		}
		// Asignar bloque
		newBlockIndex := (sb.S_first_blo - sb.S_block_start) / sb.S_block_size
		if newBlockIndex >= sb.S_blocks_count {
			return -1, nil, errors.New("error interno: S_first_blo fuera de límites al asignar nuevo bloque")
		}
		// Actualizar bitmap y SB
		err := sb.UpdateBitmapBlock(partitionPath, newBlockIndex)
		if err != nil {
			return -1, nil, fmt.Errorf("error bitmap para nuevo bloque dir %d: %w", newBlockIndex, err)
		}
		sb.S_free_blocks_count--
		sb.S_first_blo += sb.S_block_size
		// Crear, inicializar y serializar bloque vacío
		newFolderBlock := &structures.FolderBlock{}
		for i := range newFolderBlock.B_content {
			newFolderBlock.B_content[i].B_inodo = -1
		}
		newBlockOffset := int64(sb.S_block_start) + int64(newBlockIndex)*int64(sb.S_block_size)
		if err := newFolderBlock.Serialize(partitionPath, newBlockOffset); err != nil {
			return -1, nil, fmt.Errorf("falló al inicializar/serializar nuevo bloque dir %d: %w", newBlockIndex, err)
		}
		fmt.Printf("Nuevo bloque carpeta vacío asignado y serializado en índice %d\n", newBlockIndex)
		return newBlockIndex, newFolderBlock, nil // Devuelve índice Y el bloque en memoria
	}

	// Buscar en punteros directos
	for k := 0; k < 12; k++ {
		if parentInode.I_block[k] == -1 {
			fmt.Printf("Encontrado puntero directo libre en I_block[%d]. Asignando nuevo bloque carpeta...\n", k)
			newBlockIndex, newFolderBlock, err := allocateAndPrepareNewFolderBlock()
			if err != nil {
				return err
			}

			// Actualizar inodo padre para apuntar al nuevo bloque
			parentInode.I_block[k] = newBlockIndex
			currentTime := float32(time.Now().Unix())
			parentInode.I_mtime = currentTime
			parentInode.I_atime = currentTime
			if err := parentInode.Serialize(partitionPath, parentOffset); err != nil {
				return fmt.Errorf("falló al actualizar I_block[%d] del padre %d: %w", k, parentInodeIndex, err)
			}

			// Añadir entrada al nuevo bloque 
			// Usar índice 0 porque está recién creado y vacío 
			newFolderBlock.B_content[0].B_inodo = entryInodeIndex
			copy(newFolderBlock.B_content[0].B_name[:], entryName)
			newBlockOffset := int64(sb.S_block_start) + int64(newBlockIndex)*int64(sb.S_block_size)
			if err := newFolderBlock.Serialize(partitionPath, newBlockOffset); err != nil { // Sobrescribir con la entrada añadida
				return fmt.Errorf("falló al serializar nuevo bloque dir %d con la entrada: %w", newBlockIndex, err)
			}
			fmt.Printf("Nueva entrada '%s' -> %d añadida al nuevo bloque %d vía puntero directo.\n", entryName, entryInodeIndex, newBlockIndex)
			return nil
		}
	}

	// Buscar en puntero indirecto simple
	fmt.Println("Punteros directos llenos. Verificando indirección simple (I_block[12])...")
	l1Ptr := parentInode.I_block[12]
	var l1Block *structures.PointerBlock
	var l1BlockIndex int32

	if l1Ptr == -1 { // Necesitamos crear el bloque L1
		fmt.Println("I_block[12] no existe. Creando bloque de punteros L1...")
		if sb.S_free_blocks_count < 2 { // Necesitamos espacio para L1 y para el nuevo FolderBlock
			return errors.New("no hay suficientes bloques libres para crear bloque L1 y bloque de carpeta")
		}
		// Asignar bloque L1
		l1BlockIndex = (sb.S_first_blo - sb.S_block_start) / sb.S_block_size
		if l1BlockIndex >= sb.S_blocks_count {
			return errors.New("error interno: S_first_blo fuera de límites al asignar L1")
		}
		err := sb.UpdateBitmapBlock(partitionPath, l1BlockIndex)
		if err != nil {
			return fmt.Errorf("error bitmap para bloque L1 %d: %w", l1BlockIndex, err)
		}
		sb.S_free_blocks_count--
		sb.S_first_blo += sb.S_block_size

		// Actualizar inodo padre y serializarlo
		parentInode.I_block[12] = l1BlockIndex
		currentTime := float32(time.Now().Unix())
		parentInode.I_mtime = currentTime
		parentInode.I_atime = currentTime
		if err := parentInode.Serialize(partitionPath, parentOffset); err != nil {
			return fmt.Errorf("falló al actualizar I_block[12] del padre %d: %w", parentInodeIndex, err)
		}

		// Crear e inicializar bloque L1 en memoria
		l1Block = &structures.PointerBlock{}
		for i := range l1Block.P_pointers {
			l1Block.P_pointers[i] = -1
		}
		fmt.Printf("Bloque punteros L1 creado en índice %d\n", l1BlockIndex)
	} else { // El bloque L1 ya existe
		l1BlockIndex = l1Ptr
		fmt.Printf("Bloque punteros L1 ya existe en índice %d. Cargando...\n", l1BlockIndex)
		l1Block = &structures.PointerBlock{}
		l1Offset := int64(sb.S_block_start) + int64(l1BlockIndex)*int64(sb.S_block_size)
		if err := l1Block.Deserialize(partitionPath, l1Offset); err != nil {
			return fmt.Errorf("no se pudo leer bloque de punteros L1 %d existente: %w", l1BlockIndex, err)
		}
	}

	// Buscar un slot libre (-1) en el bloque L1
	foundL1PointerSlot := -1
	for k := 0; k < len(l1Block.P_pointers); k++ {
		if l1Block.P_pointers[k] == -1 {
			foundL1PointerSlot = k
			break
		}
	}

	if foundL1PointerSlot != -1 {
		fmt.Printf("Encontrado puntero libre en L1[%d]. Asignando nuevo bloque carpeta...\n", foundL1PointerSlot)
		// Asignar el nuevo bloque carpeta (ya verifica espacio libre)
		newBlockIndex, newFolderBlock, err := allocateAndPrepareNewFolderBlock()
		if err != nil {
			return err
		}

		// Actualizar el bloque L1 para que apunte al nuevo bloque
		l1Block.P_pointers[foundL1PointerSlot] = newBlockIndex
		l1Offset := int64(sb.S_block_start) + int64(l1BlockIndex)*int64(sb.S_block_size)
		if err := l1Block.Serialize(partitionPath, l1Offset); err != nil {
			return fmt.Errorf("falló al serializar bloque puntero L1 %d actualizado: %w", l1BlockIndex, err)
		}

		// Añadir entrada al nuevo bloque carpeta
		newFolderBlock.B_content[0].B_inodo = entryInodeIndex
		copy(newFolderBlock.B_content[0].B_name[:], entryName)
		newBlockOffset := int64(sb.S_block_start) + int64(newBlockIndex)*int64(sb.S_block_size)
		if err := newFolderBlock.Serialize(partitionPath, newBlockOffset); err != nil {
			return fmt.Errorf("falló al serializar nuevo bloque dir %d con la entrada: %w", newBlockIndex, err)
		}
		fmt.Printf("Nueva entrada '%s' -> %d añadida al nuevo bloque %d vía puntero indirecto simple.\n", entryName, entryInodeIndex, newBlockIndex)
		return nil
	}

	return fmt.Errorf("directorio padre (inodo %d) lleno: no hay espacio en bloques existentes ni en punteros directos/indirectos simples. Indirección doble/triple no implementada para directorios", parentInodeIndex)
}

func allocateDataBlocks(contentBytes []byte, fileSize int32, sb *structures.SuperBlock, partitionPath string) ([15]int32, error) {
	allocatedBlockIndices := [15]int32{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1} // Inicializar I_block con -1

	if fileSize == 0 {
		return allocatedBlockIndices, nil // No se necesitan bloques
	}

	blockSize := sb.S_block_size
	numBlocksNeeded := (fileSize + blockSize - 1) / blockSize

	fmt.Printf("Allocate: Necesitando %d bloques para %d bytes (tamaño bloque: %d)\n", numBlocksNeeded, fileSize, blockSize)

	directLimit := int32(12)
	simpleLimit := directLimit + 16                                      // 12 + 16 = 28
	doubleLimit := simpleLimit + 16*16                                   // 28 + 256 = 284
	tripleLimit := doubleLimit + 16*16*16                                // 284 + 4096 = 4380
	pointersPerBlock := int32(len(structures.PointerBlock{}.P_pointers)) // = 16

	if numBlocksNeeded > tripleLimit {
		return allocatedBlockIndices, fmt.Errorf("el archivo es demasiado grande (%d bloques), excede el límite de indirección triple (%d bloques)", numBlocksNeeded, tripleLimit)
	}
	if numBlocksNeeded > sb.S_free_blocks_count {
		return allocatedBlockIndices, fmt.Errorf("espacio insuficiente: se necesitan %d bloques, disponibles %d", numBlocksNeeded, sb.S_free_blocks_count)
	}

	// Variables para bloques indirectos
	var indirect1Block *structures.PointerBlock = nil // Simple
	var indirect1BlockIndex int32 = -1
	var indirect2Blocks [16]*structures.PointerBlock = [16]*structures.PointerBlock{}                               // Doble L2
	var indirect2BlockIndices [16]int32 = [16]int32{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1} // Indices L2
	var indirect2L1Block *structures.PointerBlock = nil                                                             // Doble L1
	var indirect2L1BlockIndex int32 = -1
	// Similar para triple (más complejo)

	//Bucle Principal de Asignación
	for b := int32(0); b < numBlocksNeeded; b++ {
		dataBlockIndex := (sb.S_first_blo - sb.S_block_start) / sb.S_block_size
		if dataBlockIndex >= sb.S_blocks_count {
			return allocatedBlockIndices, errors.New("error interno: S_first_blo fuera de límites al asignar bloque de datos")
		}

		// Actualizar bitmap y SB para el bloque de DATOS
		err := sb.UpdateBitmapBlock(partitionPath, dataBlockIndex)
		if err != nil {
			return allocatedBlockIndices, fmt.Errorf("error bitmap bloque datos %d: %w", dataBlockIndex, err)
		}
		sb.S_free_blocks_count--
		sb.S_first_blo += sb.S_block_size

		// Escribir datos en el bloque 
		fileBlock := &structures.FileBlock{}
		start := b * blockSize
		end := start + blockSize
		if end > fileSize {
			end = fileSize
		}
		bytesToWrite := contentBytes[start:end]
		copy(fileBlock.B_content[:], bytesToWrite)
		blockOffset := int64(sb.S_block_start) + int64(dataBlockIndex)*int64(sb.S_block_size)
		err = fileBlock.Serialize(partitionPath, blockOffset)
		if err != nil {
			return allocatedBlockIndices, fmt.Errorf("error serializando bloque datos %d: %w", dataBlockIndex, err)
		}

		// Directos (0-11)
		if b < directLimit {
			allocatedBlockIndices[b] = dataBlockIndex
			fmt.Printf("Allocate: Bloque datos %d asignado a I_block[%d]\n", dataBlockIndex, b)
			continue
		}

		// Indirecto Simple (12-27)
		if b < simpleLimit {
			idxInSimple := b - directLimit // Índice dentro del bloque de punteros simple (0-15)
			fmt.Printf("Allocate: Bloque datos %d necesita ir en Indirecto Simple (idx %d)\n", dataBlockIndex, idxInSimple)

			// Asignar el bloque de punteros L1 si es la primera vez
			if indirect1Block == nil {
				fmt.Println("Allocate: Asignando Bloque Punteros L1 (Simple)...")
				indirect1BlockIndex = (sb.S_first_blo - sb.S_block_start) / sb.S_block_size
				if indirect1BlockIndex >= sb.S_blocks_count {
					return allocatedBlockIndices, errors.New("error interno: S_first_blo fuera de límites al asignar puntero L1")
				}

				err = sb.UpdateBitmapBlock(partitionPath, indirect1BlockIndex)
				if err != nil {
					return allocatedBlockIndices, fmt.Errorf("error bitmap bloque punteros L1 %d: %w", indirect1BlockIndex, err)
				}
				sb.S_free_blocks_count-- // ¡Contar este bloque también!
				sb.S_first_blo += sb.S_block_size

				allocatedBlockIndices[12] = indirect1BlockIndex // Guardar en el inodo
				indirect1Block = &structures.PointerBlock{}     // Crear struct en memoria
				for i := range indirect1Block.P_pointers {
					indirect1Block.P_pointers[i] = -1
				}
				fmt.Printf("Allocate: Bloque Punteros L1 (Simple) asignado al índice %d\n", indirect1BlockIndex)
			}
			// Guardar puntero al bloque de datos en el struct del bloque de punteros L1
			indirect1Block.P_pointers[idxInSimple] = dataBlockIndex
			fmt.Printf("Allocate: Puntero a datos %d guardado en P_pointers[%d] del Bloque L1 (Simple)\n", dataBlockIndex, idxInSimple)
			continue
		}

		// Indirecto Doble (28-283)
		if b < doubleLimit {
			relIdxDouble := b - simpleLimit          // Índice relativo al inicio del doble indirecto (0-255)
			idxL1 := relIdxDouble / pointersPerBlock // Índice en el bloque L1 (0-15)
			idxL2 := relIdxDouble % pointersPerBlock // Índice en el bloque L2 (0-15)
			fmt.Printf("Allocate: Bloque datos %d necesita ir en Indirecto Doble (L1[%d], L2[%d])\n", dataBlockIndex, idxL1, idxL2)

			// Asignar el bloque de punteros L1 si es la primera vez para Doble
			if indirect2L1Block == nil {
				fmt.Println("Allocate: Asignando Bloque Punteros L1 (Doble)...")
				indirect2L1BlockIndex = (sb.S_first_blo - sb.S_block_start) / sb.S_block_size
				if indirect2L1BlockIndex >= sb.S_blocks_count {
					return allocatedBlockIndices, errors.New("error interno: S_first_blo fuera de límites al asignar puntero L1 doble")
				}

				err = sb.UpdateBitmapBlock(partitionPath, indirect2L1BlockIndex)
				if err != nil {
					return allocatedBlockIndices, fmt.Errorf("error bitmap bloque punteros L1 doble %d: %w", indirect2L1BlockIndex, err)
				}
				sb.S_free_blocks_count--
				sb.S_first_blo += sb.S_block_size

				allocatedBlockIndices[13] = indirect2L1BlockIndex // Guardar en el inodo
				indirect2L1Block = &structures.PointerBlock{}
				for i := range indirect2L1Block.P_pointers {
					indirect2L1Block.P_pointers[i] = -1
				}
				fmt.Printf("Allocate: Bloque Punteros L1 (Doble) asignado al índice %d\n", indirect2L1BlockIndex)
			}

			// Asignar el bloque de punteros L2 si es la primera vez para este índice L1
			if indirect2Blocks[idxL1] == nil {
				fmt.Printf("Allocate: Asignando Bloque Punteros L2 (para L1[%d])...\n", idxL1)
				blockIndexL2 := (sb.S_first_blo - sb.S_block_start) / sb.S_block_size
				if blockIndexL2 >= sb.S_blocks_count {
					return allocatedBlockIndices, errors.New("error interno: S_first_blo fuera de límites al asignar puntero L2")
				}

				err = sb.UpdateBitmapBlock(partitionPath, blockIndexL2)
				if err != nil {
					return allocatedBlockIndices, fmt.Errorf("error bitmap bloque punteros L2 %d: %w", blockIndexL2, err)
				}
				sb.S_free_blocks_count--
				sb.S_first_blo += sb.S_block_size

				indirect2L1Block.P_pointers[idxL1] = blockIndexL2   // Guardar puntero a L2 en L1
				indirect2Blocks[idxL1] = &structures.PointerBlock{} // Crear struct L2 en memoria
				indirect2BlockIndices[idxL1] = blockIndexL2         // Guardar índice L2
				for i := range indirect2Blocks[idxL1].P_pointers {
					indirect2Blocks[idxL1].P_pointers[i] = -1
				}
				fmt.Printf("Allocate: Bloque Punteros L2 asignado al índice %d (guardado en L1[%d])\n", blockIndexL2, idxL1)

				// Serializar L1 AHORA porque cambió su puntero a L2
				offsetL1 := int64(sb.S_block_start) + int64(indirect2L1BlockIndex)*int64(sb.S_block_size)
				err = indirect2L1Block.Serialize(partitionPath, offsetL1)
				if err != nil {
					return allocatedBlockIndices, fmt.Errorf("error serializando bloque puntero L1 doble %d: %w", indirect2L1BlockIndex, err)
				}
			}

			// Guardar puntero al bloque de datos en el struct del bloque de punteros L2 correspondiente
			indirect2Blocks[idxL1].P_pointers[idxL2] = dataBlockIndex
			fmt.Printf("Allocate: Puntero a datos %d guardado en P_pointers[%d] del Bloque L2 (índice %d)\n", dataBlockIndex, idxL2, indirect2BlockIndices[idxL1])
			continue
		}

		// Indirecto Triple (284 - 4379) que no haré
		if b < tripleLimit {
			return allocatedBlockIndices, fmt.Errorf("la indirección triple (bloque %d) no está implementada", b)
		}

	}

	// Serializar Bloques de Punteros Pendientes
	if indirect1Block != nil {
		fmt.Printf("Allocate: Serializando Bloque Punteros L1 (Simple) final %d\n", indirect1BlockIndex)
		offset := int64(sb.S_block_start) + int64(indirect1BlockIndex)*int64(sb.S_block_size)
		err := indirect1Block.Serialize(partitionPath, offset)
		if err != nil {
			return allocatedBlockIndices, fmt.Errorf("error serializando bloque puntero L1 simple %d: %w", indirect1BlockIndex, err)
		}
	}
	// Serializar L2 para Doble
	if indirect2L1Block != nil { // Si se usó doble indirección
		// Necesitamos serializar CADA bloque L2 que fue modificado
		for idxL1 := 0; idxL1 < len(indirect2Blocks); idxL1++ {
			if indirect2Blocks[idxL1] != nil {
				idxL2 := indirect2BlockIndices[idxL1]
				fmt.Printf("Allocate: Serializando Bloque Punteros L2 final %d (desde L1[%d])\n", idxL2, idxL1)
				offsetL2 := int64(sb.S_block_start) + int64(idxL2)*int64(sb.S_block_size)
				err := indirect2Blocks[idxL1].Serialize(partitionPath, offsetL2)
				if err != nil {
					return allocatedBlockIndices, fmt.Errorf("error serializando bloque puntero L2 %d: %w", idxL2, err)
				}
			}
		}
		// El bloque L1 ya se serializó cuando se añadieron punteros L2
	}
	return allocatedBlockIndices, nil
}
