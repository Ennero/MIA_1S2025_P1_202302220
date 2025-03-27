package reports

import (
	structures "backend/structures"
	utils "backend/utils"
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// ReporteBloque genera un reporte de los bloques y lo guarda en la ruta especificada
func ReportBlock(superblock *structures.SuperBlock, diskPath string, path string) error {
	// Crear las carpetas padre si no existen
	err := utils.CreateParentDirs(path)
	if err != nil {
		return err
	}

	// Obtener el nombre base del archivo sin la extensión
	dotFileName, outputImage := utils.GetFileNames(path)

	// Iniciar el contenido DOT
	dotContent := `digraph G {
		rankdir=LR;
		node [shape=plaintext]`

	// Iterar sobre cada inodo
	for i := int32(0); i < superblock.S_inodes_count; i++ {
		inode := &structures.Inode{}
		// Deserializar el inodo
		err := inode.Deserialize(diskPath, int64(superblock.S_inode_start+(i*superblock.S_inode_size)))
		if err != nil {
			return err
		}

		// Iterar sobre cada bloque del inodo
		for _, blockPtr := range inode.I_block {
			// Salir si el bloque no existe
			if blockPtr == -1 {
				break
			}

			// Determinar el tipo de bloque según el tipo de inodo
			switch inode.I_type[0] {
			case '0': // Bloque de carpeta
				block := &structures.FolderBlock{}
				err := block.Deserialize(diskPath, int64(superblock.S_block_start+(blockPtr*superblock.S_block_size)))
				if err != nil {
					return err
				}

				// Crear nodo para bloque de carpeta
				dotContent += fmt.Sprintf(`block%d [label=<
				<table border="0" cellborder="1" cellspacing="0">
					<tr><td colspan="2" bgcolor="lightblue"><b>Bloque Carpeta %d</b></td></tr>
					<tr><td bgcolor="lightgreen"><b>Contenido</b></td><td><b>Inodo</b></td></tr>
				`, i, blockPtr)

				// Agregar contenido de carpeta
				for _, content := range block.B_content {
					name := string(bytes.TrimRight(content.B_name[:], "\x00"))
					if name != "" {
						dotContent += fmt.Sprintf(`
							<tr><td>%s</td><td>%d</td></tr>
						`, name, content.B_inodo)
					}
				}
				dotContent += `</table>>];`

			case '1': // Bloque de archivo
				block := &structures.FileBlock{}
				err := block.Deserialize(diskPath, int64(superblock.S_block_start+(blockPtr*superblock.S_block_size)))
				if err != nil {
					return err
				}

				// Crear nodo para bloque de archivo
				content := string(bytes.TrimRight(block.B_content[:], "\x00"))
				dotContent += fmt.Sprintf(`block%d [label=<
				<table border="0" cellborder="1" cellspacing="0">
					<tr><td bgcolor="lightblue"><b>Bloque Archivo %d</b></td></tr>
					<tr><td>%s</td></tr>
				</table>>];
				`, i, blockPtr, content)

			case '2': // Bloque de apuntadores
				block := &structures.PointerBlock{}
				err := block.Deserialize(diskPath, int64(superblock.S_block_start+(blockPtr*superblock.S_block_size)))
				if err != nil {
					return err
				}

				// Crear nodo para bloque de apuntadores
				dotContent += fmt.Sprintf(`block%d [label=<
				<table border="0" cellborder="1" cellspacing="0">
					<tr><td colspan="2" bgcolor="lightblue"><b>Bloque Apuntadores %d</b></td></tr>
				`, i, blockPtr)

				// Agregar apuntadores
				for j, ptr := range block.P_pointers {
					if ptr != -1 {
						dotContent += fmt.Sprintf(`
							<tr><td>%d</td><td>%d</td></tr>
						`, j, ptr)
					}
				}
				dotContent += `</table>>];`
			}
			// Agregar flechas para conectar bloques

			// Agregar enlace al siguiente inodo si no es el último
			if i < superblock.S_blocks_count-1 {
			dotContent += fmt.Sprintf("block%d -> block%d;\n", i, i+1)
			}









			
			
		}
	}

	// Cerrar el contenido DOT
	dotContent += "}"

	// Crear el archivo DOT
	dotFile, err := os.Create(dotFileName)
	if err != nil {
		return err
	}
	defer dotFile.Close()

	// Escribir el contenido DOT en el archivo
	_, err = dotFile.WriteString(dotContent)
	if err != nil {
		return err
	}

	// Generar la imagen con Graphviz
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err = cmd.Run()
	if err != nil {
		return err
	}

	fmt.Println("Imagen de los bloques generada:", outputImage)
	return nil
}
