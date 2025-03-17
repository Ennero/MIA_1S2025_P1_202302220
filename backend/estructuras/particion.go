package estructuras

import "fmt" //para debugear xd

type Particion struct {
	part_status 		[1]byte   // Estado de la partición ('0' creada, '1' montada)
	part_type   		[1]byte   // Tipo de la partición (ej. 'P' primaria, 'E' extendida, 'L' lógica)
	part_fit    		[1]byte   // Algoritmo de ajuste ('B' Best, 'F' First, 'W' Worst)
	part_start  		int32     // Posición de inicio en el disco
	part_s   			int32     // Tamaño de la partición en bytes
	part_name   		[16]byte  // Nombre de la partición (máx. 16 caracteres)
	part_correlative	int32     // Número correlativo para identificación
	part_id  			[4]byte   // ID único de la partición (4 caracteres)
}

//Partición con los parámetros que se indiquen
//Entiendo que es como el contructor
func (p *Particion) CrearParticion (part_start, part_size int32, part_name, part_type, part_fit string) {
	p.part_status[0] = '0' // Marca la partición como creada
	p.part_start = int32(part_start)
	p.part_s = int32(part_size)

	// Asigna el tipo de partición
	if len(part_type) > 0 {
		p.part_type[0] = part_type[0]
	}

	// Asigna el ajuste de la partición
	if len(part_fit) > 0 {
		p.part_fit[0] = part_fit[0]
	}else{
		p.part_fit[0] = 'F' // Entiendo que aquí el predeterminado sería el FF entonces meh
	}
	

	// Copia el nombre de la partición a su campo
	copy(p.part_name[:], part_name)
}

func (p *Particion) MontarParticion(part_correlative int32, id string) error{
	p.part_status[0] = '1' //El valor '1' indica que la partición ha sido montada
	
	p.part_correlative = int32(part_correlative)//Asignar correlativo a la partición
	
	copy(p.part_id[:], id)// Asignar ID a la partición

	return nil
}


//Funcioncita que copio del aux para ir debuggeando
func (p *Particion) ImprimirParticion() {
	fmt.Printf("Part_status: %c\n", p.part_status[0])
	fmt.Printf("Part_type: %c\n", p.part_type[0])
	fmt.Printf("Part_fit: %c\n", p.part_fit[0])
	fmt.Printf("Part_start: %d\n", p.part_start)
	fmt.Printf("Part_size: %d\n", p.part_s)
	fmt.Printf("Part_name: %s\n", string(p.part_name[:]))
	fmt.Printf("Part_correlative: %d\n", p.part_correlative)
	fmt.Printf("Part_id: %s\n", string(p.part_id[:]))
}


