package commands

import (
	stores "backend/stores"
	structures "backend/structures"
	utils "backend/utils"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MKFILE struct {
	path string // Path del archivo
	r    bool   // Opción -r (crea directorios padres si no existen)
	size int    // Tamaño del archivo
	cont string // Contenido del archivo
}

func ParseMkfile(tokens []string) (string, error) {
	cmd := &MKFILE{} // Crea una nueva instancia de MKFILE

	//Union de tokens en una sola cadena y luego dividir por espacios, respetando las comillas
	args := strings.Join(tokens, " ")
	// Expresión regular para encontrar los parámetros del comando mkfile
	re := regexp.MustCompile(`-path=[^\s"]+|-path="[^"]+"|'[^']+'"|-r|-size=[0-9]+|-cont="[^"]+"|'[^']+'"|-cont=[^\s"]+`)
	// Encuentra todas las coincidencias de la expresión regular en la cadena de argumentos
	matches := re.FindAllString(args, -1)

	if len(matches) == 0 {
		return "", errors.New("no se reconocieron parámetros válidos")
	}
	//Verificar que todos los tokens fueron reconocidos por la expresión regular
	if len(matches) != len(tokens) {
		// Identificar el parámetro inválido
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parámetro inválido: %s", token)
			}
		}
	}

	// Itera sobre cada coincidencia encontrada
	for _, match := range matches {
		// Maneja el caso especial del parámetro -r que no tiene valor
		if match == "-r" {
			cmd.r = true
			continue
		}

		// Divide cada parte en clave y valor usando "=" como delimitador
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}

		key := strings.ToLower(kv[0])
		value := kv[1]

		// Remover comillas del valor si están presentes
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		// Switch para manejar diferentes parámetros
		switch key {
		case "-path":
			cmd.path = value
		case "-size":
			// Convierte el valor del tamaño a un entero
			size, err := strconv.Atoi(value)
			if err != nil || size < 0 {
				return "", errors.New("el tamaño debe ser un número entero no negativo")
			}
			cmd.size = size
		case "-cont":
			cmd.cont = value
		default:
			// Si el parámetro no es reconocido, devuelve un error
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Verifica que el parámetro -path haya sido proporcionado
	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	// Si no se especifica size y no se proporciona contenido, asignar tamaño predeterminado
	if cmd.size == 0 && cmd.cont == "" {
		cmd.size = 0
	}

	// Aquí se puede agregar la lógica para ejecutar el comando mkfile con los parámetros proporcionados
	err := commandMkfile(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("MKFILE: Archivo %s creado correctamente.", cmd.path), nil // Devuelve el comando MKFILE creado
}

// commandMkfile ejecuta el comando mkfile con los parámetros proporcionados
func commandMkfile(mkfile *MKFILE) error {
	// Obtener la partición montada
	partitionSuperblock, mountedPartition, partitionPath, err := stores.GetMountedPartitionSuperblock(IdPartition)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	// Preparar el contenido del archivo
	var fileContent string
	if mkfile.cont != "" {
		// Verificar si cont es una ruta a un archivo
		if _, err := os.Stat(mkfile.cont); err == nil {
			// Es una ruta a un archivo, leer su contenido
			contentBytes, err := os.ReadFile(mkfile.cont)
			if err != nil {
				return fmt.Errorf("error al leer el archivo de contenido: %w", err)
			}
			fileContent = string(contentBytes)
		} else {
			return fmt.Errorf("el archivo de contenido no existe: %w", err)
		}
	} else if mkfile.size > 0 {
		// Si se especificó tamaño pero no contenido, generar un contenido de relleno
		fileContent = strings.Repeat("1", mkfile.size)
	}

	// Crear el archivo
	err = createFile(mkfile.path, fileContent, mkfile.r, partitionSuperblock, partitionPath, mountedPartition)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}

	return nil
}

// createFile crea un nuevo archivo en el sistema de archivos con el contenido proporcionado
func createFile(filePath string, content string, createParents bool, sb *structures.SuperBlock, partitionPath string, mountedPartition *structures.Partition) error {
    // Obtener directorios padres y nombre del archivo
    parentDirs, fileName := utils.GetParentDirectories(filePath)
    parentPath := strings.Join(parentDirs, "/")

    // Crear directorios padres si no existen y la opción -r está activa
    if createParents {
        err := sb.CreateFolder(partitionPath, parentDirs, "")
        if err != nil {
            return fmt.Errorf("error al crear directorios padres: %v", err)
        }
    }

    // Buscar el inodo del directorio padre
    parentInodeNum, parentInode, err := structures.FindInodeByPath(sb, partitionPath, parentPath)
    if err != nil {
        return fmt.Errorf("directorio padre no encontrado: %v", err)
    }

    // Verificar si el archivo ya existe en el directorio padre
    exists, _ := fileExistsInDirectory(parentInode, fileName, sb, partitionPath)
    if exists {
        return errors.New("el archivo ya existe")
    }

    // Asignar un nuevo inodo para el archivo
    inodeNum, err := allocateInode(sb, partitionPath)
    if err != nil {
        return fmt.Errorf("no hay inodos libres: %v", err)
    }

    // Inicializar el inodo del archivo
    fileInode := &structures.Inode{
        I_uid:   1,
        I_gid:   1,
        I_size:  int32(len(content)),
        I_atime: float32(time.Now().Unix()),
        I_ctime: float32(time.Now().Unix()),
        I_mtime: float32(time.Now().Unix()),
        I_type:  [1]byte{'1'},
        I_perm:  [3]byte{'6', '6', '4'},
    }

    // Asignar bloques y escribir contenido
    chunks := utils.SplitStringIntoChunks(content)
    var blocks []int32
    for _, chunk := range chunks {
        blockNum, err := allocateBlock(sb, partitionPath)
        if err != nil {
            return fmt.Errorf("no hay bloques libres: %v", err)
        }
        blocks = append(blocks, blockNum)

        // Escribir bloque de archivo
        fileBlock := &structures.FileBlock{}
        copy(fileBlock.B_content[:], chunk)
        blockOffset := int64(sb.S_block_start + blockNum*sb.S_block_size)
        if err := fileBlock.Serialize(partitionPath, blockOffset); err != nil {
            return err
        }
    }

    // Asignar bloques al inodo
    for i, block := range blocks {
        if i < 12 {
            fileInode.I_block[i] = block
        } else {
            // Manejar bloques indirectos (implementación básica)
            return errors.New("los bloques indirectos no están implementados")
        }
    }

    // Escribir inodo del archivo
    inodeOffset := int64(sb.S_inode_start + inodeNum*sb.S_inode_size)
    if err := fileInode.Serialize(partitionPath, inodeOffset); err != nil {
        return err
    }

    // Agregar entrada al directorio padre
    if err := addDirectoryEntry(parentInode, parentInodeNum, fileName, inodeNum, sb, partitionPath); err != nil {
        return fmt.Errorf("error al agregar entrada al directorio: %v", err)
    }

    // Actualizar superbloque
    sb.S_free_inodes_count--
    sb.S_free_blocks_count -= int32(len(blocks))
    if err := sb.Serialize(partitionPath, int64(mountedPartition.Part_start)); err != nil {
        return err
    }

    return nil
}

// Funciones auxiliares necesarias
func allocateInode(sb *structures.SuperBlock, path string) (int32, error) {
    file, err := os.OpenFile(path, os.O_RDWR, 0644)
    if err != nil {
        return -1, err
    }
    defer file.Close()

    for i := int32(0); i < sb.S_inodes_count; i++ {
        offset := sb.S_bm_inode_start + i
        file.Seek(int64(offset), 0)
        bit := make([]byte, 1)
        file.Read(bit)
        
        if bit[0] == '0' {
            file.WriteAt([]byte{'1'}, int64(offset))
            return i, nil
        }
    }
    return -1, errors.New("no hay inodos disponibles")
}

func allocateBlock(sb *structures.SuperBlock, path string) (int32, error) {
    file, err := os.OpenFile(path, os.O_RDWR, 0644)
    if err != nil {
        return -1, err
    }
    defer file.Close()

    for i := int32(0); i < sb.S_blocks_count; i++ {
        offset := sb.S_bm_block_start + i
        file.Seek(int64(offset), 0)
        bit := make([]byte, 1)
        file.Read(bit)
        
        if bit[0] == '0' {
            file.WriteAt([]byte{'1'}, int64(offset))
            return i, nil
        }
    }
    return -1, errors.New("no hay bloques disponibles")
}

func addDirectoryEntry(parentInode *structures.Inode, parentInodeNum int32, name string, targetInode int32, sb *structures.SuperBlock, path string) error {
    entry := structures.FolderContent{
        B_inodo: targetInode,
    }
    copy(entry.B_name[:], name)

    for i, blockPtr := range parentInode.I_block {
        if blockPtr == -1 {
            // Asignar nuevo bloque
            newBlock, err := allocateBlock(sb, path)
            if err != nil {
                return err
            }
            parentInode.I_block[i] = newBlock
            
            // Inicializar nuevo bloque
            newFolderBlock := structures.FolderBlock{}
            newFolderBlock.B_content[0] = entry
            for j := 1; j < 4; j++ {
                newFolderBlock.B_content[j].B_inodo = -1
            }
            
            // Escribir bloque
            blockOffset := int64(sb.S_block_start + newBlock*sb.S_block_size)
            if err := newFolderBlock.Serialize(path, blockOffset); err != nil {
                return err
            }
            
            // Actualizar inodo padre
            parentInode.I_size += sb.S_block_size
            parentInode.I_mtime = float32(time.Now().Unix())
            inodeOffset := int64(sb.S_inode_start + parentInodeNum*sb.S_inode_size)
            return parentInode.Serialize(path, inodeOffset)
        }

        // Leer bloque existente
        folderBlock := structures.FolderBlock{}
        blockOffset := int64(sb.S_block_start + blockPtr*sb.S_block_size)
        if err := folderBlock.Deserialize(path, blockOffset); err != nil {
            return err
        }

        // Buscar espacio
        for j, e := range folderBlock.B_content {
            if e.B_inodo == -1 {
                folderBlock.B_content[j] = entry
                if err := folderBlock.Serialize(path, blockOffset); err != nil {
                    return err
                }
                // Actualizar inodo padre
                parentInode.I_mtime = float32(time.Now().Unix())
                inodeOffset := int64(sb.S_inode_start + parentInodeNum*sb.S_inode_size)
                return parentInode.Serialize(path, inodeOffset)
            }
        }
    }
    return errors.New("directorio lleno")
}

func fileExistsInDirectory(parentInode *structures.Inode, name string, sb *structures.SuperBlock, path string) (bool, error) {
    for _, blockPtr := range parentInode.I_block {
        if blockPtr == -1 {
            continue
        }
        
        folderBlock := structures.FolderBlock{}
        blockOffset := int64(sb.S_block_start + blockPtr*sb.S_block_size)
        if err := folderBlock.Deserialize(path, blockOffset); err != nil {
            return false, err
        }
        
        for _, entry := range folderBlock.B_content {
            if entry.B_inodo != -1 && strings.TrimRight(string(entry.B_name[:]), "\x00") == name {
                return true, nil
            }
        }
    }
    return false, nil
}