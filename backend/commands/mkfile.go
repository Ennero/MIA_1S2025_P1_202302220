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
	// 1. Obtener Autenticación y Partición Montada
	var userID int32 = 1  // ID del usuario logueado (POR DEFECTO 1 = root, OBTENER DE stores.Auth)
	var groupID int32 = 1 // ID del grupo del usuario (POR DEFECTO 1 = root, OBTENER DE stores.Auth O users.txt)
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

	// 2. Limpiar Path y Obtener Padre/Nombre
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
		return err // Error ya viene formateado desde ensureParentDirExists
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
			// Size es 0, contenido vacío
			contentBytes = []byte{}
		}
	}
	fmt.Printf("Tamaño final del archivo: %d bytes\n", fileSize)


	// Calcular bloques necesarios y validar límite (Directos por ahora)
	blockSize := partitionSuperblock.S_block_size
	numBlocksNeeded := int32(0)
	if fileSize > 0 {
		numBlocksNeeded = (fileSize + blockSize - 1) / blockSize // Ceiling division
	}

	// Simplificación: Limitar a bloques directos (0-11) por ahora
	maxDirectBlocks := int32(12)
	if numBlocksNeeded > maxDirectBlocks {
		return fmt.Errorf("la creación de archivos con más de %d bloques (requiere %d) no está implementada (sin bloques indirectos)", maxDirectBlocks, numBlocksNeeded)
	}
	// Validar si hay suficientes bloques libres
	if numBlocksNeeded > partitionSuperblock.S_free_blocks_count {
		return fmt.Errorf("espacio insuficiente en disco: se necesitan %d bloques, disponibles %d", numBlocksNeeded, partitionSuperblock.S_free_blocks_count)
	}
	// Validar si hay suficientes inodos libres
	if partitionSuperblock.S_free_inodes_count < 1 {
		return errors.New("espacio insuficiente: no hay inodos libres disponibles")
	}

	// Asignar Bloques de Datos
	fmt.Printf("Asignando %d bloque(s) de datos...\n", numBlocksNeeded)
	allocatedBlockIndices := make([]int32, 15) // Array para I_block
	for i := range allocatedBlockIndices {
		allocatedBlockIndices[i] = -1
	} // Inicializar con -1

	for b := int32(0); b < numBlocksNeeded; b++ {
		// Calcular índice del próximo bloque libre
		newBlockIndex := (partitionSuperblock.S_first_blo - partitionSuperblock.S_block_start) / partitionSuperblock.S_block_size
		if newBlockIndex >= partitionSuperblock.S_blocks_count {
			return errors.New("error interno: S_first_blo apunta fuera de los límites")
		}
		// Asignar al array del inodo (directos primero)
		if b < maxDirectBlocks {
			allocatedBlockIndices[b] = newBlockIndex
		} // else: lógica para indirectos iría aquí

		err = partitionSuperblock.UpdateBitmapBlock(partitionPath, newBlockIndex)
		if err != nil {
			return fmt.Errorf("error actualizando bitmap para bloque %d: %w", newBlockIndex, err)
		}
		partitionSuperblock.S_free_blocks_count--
		partitionSuperblock.S_first_blo += partitionSuperblock.S_block_size

		// Preparar y escribir datos en el bloque
		fileBlock := &structures.FileBlock{} // Crear bloque en blanco
		start := b * blockSize
		end := start + blockSize
		if end > fileSize {
			end = fileSize
		}
		bytesToWrite := contentBytes[start:end]
		copy(fileBlock.B_content[:], bytesToWrite) // Copiar datos al bloque

		// Serializar bloque de datos
		blockOffset := int64(partitionSuperblock.S_block_start) + int64(newBlockIndex)*int64(partitionSuperblock.S_block_size)
		err = fileBlock.Serialize(partitionPath, blockOffset)
		if err != nil {
			return fmt.Errorf("error serializando bloque de datos %d: %w", newBlockIndex, err)
		}
	}

	fmt.Println("Asignando inodo...")
	newInodeIndex := (partitionSuperblock.S_first_ino - partitionSuperblock.S_inode_start) / partitionSuperblock.S_inode_size
	if newInodeIndex >= partitionSuperblock.S_inodes_count {
		return errors.New("error interno: S_first_ino apunta fuera de los límites")
	}

	// Actualizar bitmap y superbloque (inodo)
	err = partitionSuperblock.UpdateBitmapInode(partitionPath, newInodeIndex)
	if err != nil {
		return fmt.Errorf("error actualizando bitmap para inodo %d: %w", newInodeIndex, err)
	}
	partitionSuperblock.S_free_inodes_count--
	partitionSuperblock.S_first_ino += partitionSuperblock.S_inode_size

	//Crear y Serializar Estructura Inodo
	currentTime := float32(time.Now().Unix())
	newInode := &structures.Inode{
		I_uid:   userID,
		I_gid:   groupID,
		I_size:  fileSize,
		I_atime: currentTime,
		I_ctime: currentTime,
		I_mtime: currentTime,
		I_type:  [1]byte{'1'},           // Tipo Archivo
		I_perm:  [3]byte{'6', '6', '4'}, // Permisos rw-rw-r--
	}
	copy(newInode.I_block[:], allocatedBlockIndices) // Copiar punteros de bloque asignados

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

	//Serializar Superbloque (Importante hacerlo al final)
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
		// Raíz siempre existe (inodo 0), obtenerlo
		inode := &structures.Inode{}
		offset := int64(sb.S_inode_start) // Raíz es inodo 0
		err := inode.Deserialize(partitionPath, offset)
		if err != nil {
			return -1, nil, fmt.Errorf("error crítico: no se pudo deserializar inodo raíz (0): %w", err)
		}
		if inode.I_type[0] != '0' {
			return -1, nil, errors.New("error crítico: inodo raíz (0) no es un directorio")
		}
		return 0, inode, nil // Retorna índice 0 y el inodo raíz
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

	// Padre no encontrado (o hubo otro error)
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

// Retorna si existe, el índice del inodo encontrado y su tipo (0=dir, 1=archivo)
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
					return // Encontrado, salir
				}
			}
		}
	}
	return
}

// Busca un slot libre
func addEntryToParent(parentInodeIndex int32, entryName string, entryInodeIndex int32, sb *structures.SuperBlock, partitionPath string) error {

	parentInode := &structures.Inode{}
	parentOffset := int64(sb.S_inode_start) + int64(parentInodeIndex)*int64(sb.S_inode_size)
	if err := parentInode.Deserialize(partitionPath, parentOffset); err != nil {
		return fmt.Errorf("no se pudo leer inodo padre %d para añadir entrada: %w", parentInodeIndex, err)
	}

	// Buscar un slot libre en los bloques existentes del padre
	for _, blockPtr := range parentInode.I_block {
		if blockPtr == -1 {
			continue
		}
		if blockPtr < 0 || blockPtr >= sb.S_blocks_count {
			continue
		} 

		folderBlock := &structures.FolderBlock{}
		blockOffset := int64(sb.S_block_start) + int64(blockPtr)*int64(sb.S_block_size)
		if err := folderBlock.Deserialize(partitionPath, blockOffset); err != nil {
			fmt.Printf("Advertencia: No se pudo leer bloque %d del padre %d para añadir entrada\n", blockPtr, parentInodeIndex)
			continue 
		}

		for i := 0; i < 4; i++ {
			if folderBlock.B_content[i].B_inodo == -1 {

				fmt.Printf("Encontrado slot libre %d en bloque %d del padre %d\n", i, blockPtr, parentInodeIndex)
				folderBlock.B_content[i].B_inodo = entryInodeIndex
				copy(folderBlock.B_content[i].B_name[:], entryName)

				if err := folderBlock.Serialize(partitionPath, blockOffset); err != nil {
					return fmt.Errorf("falló al escribir la nueva entrada en el bloque %d: %w", blockPtr, err)
				}

				// Actualizar tiempo de modificación y acceso del inodo padre
				currentTime := float32(time.Now().Unix())
				parentInode.I_mtime = currentTime
				parentInode.I_atime = currentTime
				if err := parentInode.Serialize(partitionPath, parentOffset); err != nil {
					return fmt.Errorf("falló al actualizar tiempos del inodo padre %d: %w", parentInodeIndex, err)
				}

				return nil 
			}
		}
	}
	return fmt.Errorf("no se encontró espacio en los bloques existentes del directorio padre (inodo %d)", parentInodeIndex)
}

