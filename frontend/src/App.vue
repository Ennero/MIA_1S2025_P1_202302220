<template>
  <div id="app" class="container mt-5">
    <h1 class="text-center mb-4">Sistema de archivos EXT2</h1>
    
    <div class="card p-4 shadow-sm">
      <div class="mb-3">
        <label class="form-label">Ingresa el comando o carga un script:</label>
        <textarea v-model="entrada" class="form-control" rows="4" placeholder="Escribe aquí..."></textarea>
      </div>
      
      <div class="mb-3 d-flex justify-content-between">
        <input type="file" class="form-control" @change="handleFileUpload"/>
        <button class="btn btn-primary ms-2" @click="ejecutar">Ejecutar</button>
        <button class="btn btn-danger ms-2" @click="limpiar">Limpiar</button>
      </div>
    </div>
    
    <div class="card p-4 mt-4 shadow-sm">
      <label class="form-label">Salida de comandos:</label>
      <textarea v-model="salida" class="form-control bg-light" rows="6" readonly></textarea>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      entrada: "",
      salida: "",
    };
  },
  methods: {
    handleFileUpload(event) {
      const file = event.target.files[0];
      if (file) {
        const reader = new FileReader();
        reader.onload = (e) => {
          this.entrada = e.target.result;
        };
        reader.readAsText(file);
      }
    },
    ejecutar() {
      //Lo que se manda al backend iria aquí
      this.salida = "Ejecutando comandos...\n";
      //lo que se obtiene en el output

      setTimeout(() => {
        this.salida += "Comando ejecutado con éxito.\n";
      }, 1000);
    },
    limpiar() {
      this.entrada = "";
      this.salida = "";
    },
  },
};
</script>

<style>
@import "https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css";
#app {
  max-width: 700px;
  margin: auto;
}
</style>
