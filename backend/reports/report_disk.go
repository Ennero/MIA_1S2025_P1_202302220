package reports

import (
	structures "backend/structures"
	utils "backend/utils"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ReportDisk(mbr *structures.MBR, diskPath string, outputPath string) error {
	err := utils.CreateParentDirs(outputPath)
	if err != nil {
		return err
	}
	totalSize := mbr.Mbr_size
	name := utils.GetDiskName(diskPath)

	dotFileName, outputImage := utils.GetFileNames(outputPath)

	content := "digraph G {\n"
	content += "\tnode [shape=none];\n"
	content += "\tgraph [splines=false];\n"
	content += "\tsubgraph cluster_disk {\n"
	content += fmt.Sprintf("\t\tlabel=\"%s\";\n", name)
	content += "\t\tstyle=filled;\n"
	content += "\t\tfillcolor=white;\n"
	content += "\t\tcolor=black;\n"
	content += "\t\tpenwidth=2;\n"

	content += "\t\ttable [label=<\n\t\t\t<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\" CELLPADDING=\"15\" WIDTH=\"800\">\n"
	content += "\t\t\t<TR>\n"
	content += "\t\t\t<TD BGCOLOR=\"gray\" ALIGN=\"CENTER\"><B>MBR</B></TD>\n"

	var usedSpace int32 = 0

	for i := 0; i < 4; i++ {
		part := mbr.Mbr_partitions[i]

		if part.Part_size > 0 {
			percentage := float64(part.Part_size) / float64(totalSize) * 100
			partName := strings.TrimRight(string(part.Part_name[:]), "\x00")
			cellWidth := int(percentage * 8) // Ajustamos el ancho en base al porcentaje

			switch part.Part_type[0] {
			case 'P':
				content += fmt.Sprintf("\t\t\t<TD BGCOLOR=\"lightblue\" WIDTH=\"%d\" ALIGN=\"CENTER\"><B>Primaria</B><BR/><B>%s</B><BR/>%.2f%% del disco</TD>\n", 
					cellWidth, partName, percentage)
				usedSpace += part.Part_size
			case 'E':
				// La celda en sí es la extendida, sin tabla anidada adicional
				content += fmt.Sprintf("\t\t\t<TD BGCOLOR=\"orange\" WIDTH=\"%d\" ALIGN=\"CENTER\" CELLPADDING=\"0\">\n", cellWidth)
				content += "\t\t\t\t<TABLE BORDER=\"0\" CELLBORDER=\"0\" CELLSPACING=\"0\" CELLPADDING=\"5\" WIDTH=\"100%\">\n"
				content += "\t\t\t\t<TR><TD COLSPAN=\"100\" ALIGN=\"CENTER\"><B>Extendida</B></TD></TR>\n"
				content += "\t\t\t\t<TR>\n"

				file, err := os.Open(diskPath)
				if err != nil {
					return fmt.Errorf("error abriendo el archivo del disco: %v", err)
				}
				defer file.Close()

				var ebr structures.EBR
				offset := part.Part_start
				logicalCount := 0

				for {
					file.Seek(int64(offset), os.SEEK_SET)
					err := binary.Read(file, binary.LittleEndian, &ebr)
					if err != nil || ebr.Part_size <= 0 {
						break
					}

					logicalCount++
					logicalPercentage := float64(ebr.Part_size) / float64(totalSize) * 100
					logicalName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")
					
					// EBR ahora es gris
					content += "\t\t\t\t<TD BGCOLOR=\"gray\" ALIGN=\"CENTER\" BORDER=\"1\"><B>EBR</B></TD>\n"
					content += fmt.Sprintf("\t\t\t\t<TD BGCOLOR=\"lightgreen\" ALIGN=\"CENTER\" BORDER=\"1\"><B>Lógica</B><BR/>%s<BR/>%.2f%%</TD>\n", 
						logicalName, logicalPercentage)
					
					usedSpace += ebr.Part_size

					if ebr.Part_next <= 0 || ebr.Part_next >= mbr.Mbr_size {
						break
					}
					offset = ebr.Part_next
				}

				content += "\t\t\t\t<TD BGCOLOR=\"gray\" ALIGN=\"CENTER\" BORDER=\"1\"><B>EBR</B></TD>\n"
				content += "\t\t\t\t</TR>\n"
				content += "\t\t\t\t</TABLE>\n"
				content += "\t\t\t</TD>\n"
			}
		}
	}

	freeSpace := totalSize - usedSpace
	freePercentage := float64(freeSpace) / float64(totalSize) * 100
	freeWidth := int(freePercentage * 8) // Ajustamos el ancho en base al porcentaje

	if freeSpace > 0 {
		content += fmt.Sprintf("\t\t\t<TD BGCOLOR=\"#F5F5F5\" WIDTH=\"%d\" ALIGN=\"CENTER\"><B>Libre</B><BR/>%.2f%% del disco</TD>\n", 
			freeWidth, freePercentage)
	}
	
	content += "\t\t\t</TR>\n"
	content += "\t\t\t</TABLE>\n>];\n"
	content += "\t}\n"
	content += "}\n"

	// Guardar el contenido DOT en un archivo
	file, err := os.Create(dotFileName)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}

	// Ejecutar el comando Graphviz para generar la imagen
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Reporte DISK generado:", outputImage)
	return nil
}