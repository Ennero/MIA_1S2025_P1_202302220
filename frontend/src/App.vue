<template>
  <div id="app" class="container py-5">
    <div class="row justify-content-center">
      <div class="col-md-10">
        <!-- Cabecera -->
        <div class="text-center mb-4">
          <h1 class="display-5 fw-bold text-primary">Sistema de archivos EXT2</h1>
          <p class="lead text-secondary">Gestión de comandos y scripts</p>
          <hr class="my-4 text-primary opacity-75">
        </div>
        
        <!-- Panel de comandos -->
        <div class="card border-0 shadow-lg mb-4 bg-white rounded">
          <div class="card-header bg-primary text-white p-3">
            <div class="d-flex align-items-center">
              <i class="bi bi-terminal-fill me-2 fs-5"></i>
              <h5 class="mb-0">Consola de comandos</h5>
            </div>
          </div>
          <div class="card-body p-4">
            <div class="form-floating mb-3">
              <textarea 
                v-model="entrada" 
                class="form-control bg-light"
                id="commandTextarea" 
                style="height: 120px"
                placeholder="Escribe comandos aquí..."
              ></textarea>
              <label for="commandTextarea">Ingresa el comando o script</label>
            </div>
            
            <div class="row g-3">
              <div class="col-md-6">
                <div class="input-group">
                  <label class="input-group-text bg-secondary text-white">
                    <i class="bi bi-file-earmark-text"></i>
                  </label>
                  <input 
                    type="file" 
                    class="form-control" 
                    @change="handleFileUpload"
                    accept=".mias"
                    id="fileInput"
                  />
                </div>
                <div class="form-text text-muted mt-1">
                  <i class="bi bi-info-circle-fill me-1"></i> Solo archivos con extensión .mias
                </div>
                <div v-if="fileError" class="alert alert-danger mt-2 py-2 small">
                  <i class="bi bi-exclamation-triangle-fill me-1"></i> {{ fileError }}
                </div>
              </div>
              <div class="col-md-3">
                <button class="btn btn-primary w-100 d-flex justify-content-center align-items-center" @click="ejecutar">
                  <i class="bi bi-play-fill me-2"></i> Ejecutar
                </button>
              </div>
              <div class="col-md-3">
                <button class="btn btn-danger w-100 d-flex justify-content-center align-items-center" @click="limpiar">
                  <i class="bi bi-trash me-2"></i> Limpiar
                </button>
              </div>
            </div>
          </div>
        </div>
        
        <!-- Panel de salida -->
        <div class="card border-0 shadow-lg bg-white rounded">
          <div class="card-header bg-success text-white p-3">
            <div class="d-flex align-items-center">
              <i class="bi bi-code-square me-2 fs-5"></i>
              <h5 class="mb-0">Resultado de comandos</h5>
            </div>
          </div>
          <div class="card-body p-4">
            <!-- Se eliminó form-floating y su label -->
            <textarea 
              v-model="salida" 
              class="form-control bg-dark text-light font-monospace"
              style="height: 180px"
              id="outputTextarea"
              readonly
              placeholder="La salida aparecerá aquí..."
            ></textarea>
          </div>
          <div class="card-footer bg-light p-3 text-end">
            <span class="badge bg-info text-dark">
              <i class="bi bi-info-circle me-1"></i> Sistema de archivos EXT2 • Enner Mendizabal 202302220
            </span>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      entrada: "",
      salida: "",
      fileError: ""
    };
  },
  methods: {
    handleFileUpload(event) {
      const file = event.target.files[0];
      this.fileError = "";
      
      if (!file) return;
      
      // Verificar si la extensión del archivo es .mias
      const fileName = file.name;
      const fileExtension = fileName.split('.').pop().toLowerCase();
      
      if (fileExtension !== 'mias') {
        this.fileError = "Solo se permiten archivos con extensión .mias";
        // Limpiar la selección del input file
        event.target.value = '';
        return;
      }
      
      // Si pasa la validación, proceder con la carga
      const reader = new FileReader();
      reader.onload = (e) => {
        this.entrada = e.target.result;
        this.salida = `✅ Archivo cargado: ${fileName}`;
      };
      reader.readAsText(file);
    },
    async ejecutar() {
      if(!this.entrada.trim()) {
        this.salida = "⚠️ No hay comandos para ejecutar";
        return;
      }
      
      this.salida = "🔄 Ejecutando comandos...\n";

      try {
        const response = await fetch('http://localhost:3001/', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({command: this.entrada}),
        });
        
        if(!response.ok) {
          throw new Error('Error al ejecutar el comando');
        }

        const data = await response.json();
        this.salida += data.output;
      } catch (error) {
        this.salida += "❌ Error al ejecutar el comando.\n";
      }
    },
    limpiar() {
      this.entrada = "";
      this.salida = "";
      this.fileError = "";
      // Limpiar también el input de archivo
      const fileInput = document.getElementById('fileInput');
      if (fileInput) fileInput.value = '';
    },
  },
};
</script>

<style>
@import "https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css";
@import "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.10.0/font/bootstrap-icons.css";

body {
  background: #f8f9fa;
}
.card {
  transition: transform 0.2s;
}

.card:hover {
  transform: translateY(-5px);
}

textarea.form-control:focus {
  box-shadow: 0 0 0 0.25rem rgba(13, 110, 253, 0.25);
  border-color: #86b7fe;
}

.font-monospace {
  font-family: SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}
</style>