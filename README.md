<div align="center">

# 🗄️ EXT2 File System Simulator & Virtual Disk Manager

**Simulador integral de particionamiento de discos virtuales y sistema de archivos EXT2**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Vue.js](https://img.shields.io/badge/Vue.js-3.x-4FC08D?style=for-the-badge&logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![University](https://img.shields.io/badge/USAC-Ingenier%C3%ADa%20en%20Sistemas-003366?style=for-the-badge)](https://ingenieria.usac.edu.gt/)

</div>

---

## 📌 Descripción General

Este proyecto es una implementación completa de un **simulador de discos virtuales y sistema de archivos tipo UNIX (EXT2)**, desarrollado con una arquitectura desacoplada Cliente-Servidor:
- **Backend:** Desarrollado en **Go (Golang)**, responsable de la manipulación binaria a bajo nivel, serialización/deserialización de estructuras de disco en archivos `.dsk`, análisis sintáctico de comandos y generación de reportes gráficos con Graphviz.
- **Frontend:** Interfaz web interactiva construida en **Vue.js**, que permite redactar scripts, ejecutar comandos, monitorear el estado de las particiones y visualizar los reportes del sistema de archivos en tiempo real.

Desarrollado en el marco del curso **Manejo e Implementación de Archivos (MIA)** en la Universidad de San Carlos de Guatemala.

---

## 🚀 Características Principales

### 💾 1. Gestión de Discos y Particiones
- **Discos Virtuales:** Creación (`mkdisk`) y eliminación (`rmdisk`) de archivos binarios que actúan como discos físicos, con algoritmos de ajuste:
  - `BF` (Best Fit)
  - `FF` (First Fit)
  - `WF` (Worst Fit)
- **Particionamiento (`fdisk`):**
  - **Particiones Primarias:** Hasta 4 particiones primarias por disco.
  - **Particiones Extendidas:** Soporte para 1 partición extendida por disco.
  - **Particiones Lógicas:** Múltiples particiones lógicas dentro de la extendida gestionadas mediante lista enlazada de `EBR` (Extended Boot Record).
- **Montaje (`mount` / `unmount`):** Sistema de montaje dinámico con identificadores alfanuméricos únicos por partición (`carnet + número + letra`).

### 📁 2. Sistema de Archivos EXT2
- **Formateo (`mkfs`):** Cálculo dinámico de inodos y bloques según la fórmula estándar de EXT2, inicialización de SuperBloque, Bitmap de Inodos, Bitmap de Bloques, Inodo Raíz (`/`) y archivo inicial `/users.txt`.
- **Estructura de Inodos y Bloques:**
  - Inodos de Archivo y Carpeta.
  - Bloques de Carpetas con apuntadores directos (`.`, `..` y 2 entradas más).
  - Bloques de Datos con capacidad de 64 bytes por bloque.
  - Apuntadores directos, indirectos simples, indirectos dobles e indirectos triples.
- **Gestión de Sesiones y Permisos:**
  - Autenticación (`login` / `logout`).
  - Creación y eliminación de grupos (`mkgrp` / `rmgrp`).
  - Creación y eliminación de usuarios (`mkusr` / `rmusr`).
  - Permisos UGO (User, Group, Others) en lectura, escritura y ejecución.
- **Operaciones de Archivos y Carpetas:**
  - Creación jerárquica de carpetas (`mkdir` con bandera recursiva `-r`).
  - Creación de archivos de texto (`mkfile` con `-r` y `-size`).
  - Lectura de archivos (`cat`).

### 📊 3. Generación de Reportes Gráficos (Graphviz)
- Reporte de **MBR** y **EBR** en formato tabla visual.
- Reporte del **SuperBloque**.
- Reporte de **Inodos y Bloques** navegable.
- Visualización de **Bitmaps** (Inodos y Bloques).
- Reporte de árbol de directorios (**Tree Report**).

---

## 🏗️ Arquitectura del Sistema

```
MIA_1S2025_P1_202302220/
├── backend/                  # Servidor API y Lógica de Bajo Nivel (Go)
│   ├── analyzer/             # Lexer y parser de comandos
│   ├── commands/             # Implementación de cada comando (mkdisk, fdisk, mkfs, etc.)
│   ├── disks/                # Archivos binarios de discos simulados (.dsk)
│   ├── reports/              # Generadores de código DOT y render Graphviz
│   ├── stores/               # Estado en memoria (particiones montadas, sesión activa)
│   ├── structures/           # Estructuras binarias (MBR, EBR, Superblock, Inode, Block)
│   ├── utils/                # Utilidades de conversión binaria y serialización
│   └── main.go               # Punto de entrada HTTP y CLI
├── frontend/                 # Interfaz Web (Vue.js)
│   ├── src/                  # Componentes Vue, vistas y stores
│   └── package.json          # Dependencias de npm
├── doc/                      # Documentación completa y Manual Técnico
│   └── manualTecnico.mkd     # Manual técnico detallado (~75 KB)
├── LICENSE                   # Licencia MIT
└── README.md                 # Este documento
```

---

## 💻 Requisitos Previos

- **Go:** Versión `1.20` o superior.
- **Node.js:** Versión `18.x` o superior y `npm`.
- **Graphviz:** Herramienta `dot` instalada en el sistema para renderizar imágenes de reportes (`sudo apt install graphviz` o vía `winget install Graphviz.Graphviz` en Windows).

---

## ⚙️ Instalación y Ejecución

### 1. Iniciar el Backend (Go)

```bash
cd backend
go mod download
go run main.go
```
*El servidor backend iniciará y quedará a la escucha de peticiones HTTP en el puerto configurado (ej. `http://localhost:8080`).*

### 2. Iniciar el Frontend (Vue.js)

En otra terminal:

```bash
cd frontend
npm install
npm run serve
```
*Abre tu navegador en `http://localhost:8080` (o el puerto indicado por Vue CLI).*

---

## 📝 Ejemplo de Script de Comandos

```bash
# 1. Crear un disco de 10 MB con ajuste First Fit
mkdisk -size=10 -unit=M -fit=FF

# 2. Crear una partición primaria de 5 MB
fdisk -size=5 -unit=M -driveletter=A -name=Particion1 -type=P

# 3. Montar la partición
mount -driveletter=A -name=Particion1

# 4. Formatear la partición con EXT2
mkfs -id=20A1 -type=full

# 5. Iniciar sesión como root
login -user=root -pass=123 -id=20A1

# 6. Crear directorios y archivos
mkdir -path=/home/archivos -r
mkfile -path=/home/archivos/saludo.txt -size=50 -cont="Hola mundo desde EXT2 en Go"
```

---

## 📄 Licencia

Este proyecto está bajo la Licencia **MIT**. Consulta el archivo [LICENSE](LICENSE) para más detalles.

---

## 👨‍💻 Autor

- **Enner Mendizabal** - [@Ennero](https://github.com/Ennero)  
- *Facultad de Ingeniería, Universidad de San Carlos de Guatemala.*
