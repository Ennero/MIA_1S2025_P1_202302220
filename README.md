# Simulador EXT2 y administrador de discos virtuales

Proyecto académico de Manejo e Implementación de Archivos. El backend Go interpreta comandos para discos virtuales, particiones, usuarios y archivos; el frontend Vue 3 permite enviarlos y consultar resultados. Los reportes usan Graphviz.

## Requisitos

- Go 1.22.2 o superior, según backend/go.mod.
- Node.js/npm compatibles con Vue CLI 5 y el lockfile incluido.
- Graphviz instalado como programa del sistema (`dot` en PATH).

## Arranque

Desde la raíz:

```bash
cd backend
go mod download
go run .
```

La API escucha en `http://localhost:3001`. En otra terminal desde la raíz:

```bash
cd frontend
npm ci
npm run serve
```

Abrir la URL de Vue CLI, habitualmente `http://localhost:8080`. La API recibe POST `/` con un JSON cuyo campo `command` contiene las instrucciones.

## Primera demostración

En el editor del frontend, usar una ruta de disco de prueba:

```text
mkdisk -size=10 -unit=M -fit=FF -path=./disks/demo.mia
fdisk -size=5 -unit=M -path=./disks/demo.mia -name=Particion1 -type=P
mount -path=./disks/demo.mia -name=Particion1
```

El montaje devuelve un identificador. Utilizar **ese valor** para los comandos mkfs/login y reportes, siguiendo el [manual técnico](doc/manualTecnico.mkd). No copiar un ID de otro entorno. En mkfile, `-cont` representa la ruta de un archivo de contenido, no un literal de texto.

## Organización

| Ruta | Responsabilidad |
|---|---|
| [backend/commands/](backend/commands/) | Discos, particiones, sesiones, usuarios y operaciones de archivos. |
| [backend/structures/](backend/structures/) | Estructuras binarias del sistema simulado. |
| [backend/reports/](backend/reports/) | Reportes y generación de diagramas. |
| [frontend/](frontend/) | Interfaz Vue. |
| [doc/manualTecnico.mkd](doc/manualTecnico.mkd) | Diseño, comandos y procedimientos de la entrega. |

## Alcance

Es un simulador académico de EXT2 en archivos `.mia`, no una herramienta para modificar sistemas de archivos de un disco físico. Formatear y eliminar discos afecta los archivos simulados indicados: usar copias de prueba. Los scripts históricos pueden contener rutas del autor y deben adaptarse antes de ejecutarlos.

## Licencia

Consultar [LICENSE](LICENSE). Se conservan los créditos de la entrega y los avisos propios de las dependencias.
