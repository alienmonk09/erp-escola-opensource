// main.js — hello vazio da fundação (F-01). Só exibe o status do backend.
fetch("/api/saude")
  .then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.status))))
  .then((j) => {
    document.getElementById("saude").textContent = JSON.stringify(j);
  })
  .catch(() => {
    document.getElementById("saude").textContent = "indisponível";
  });
