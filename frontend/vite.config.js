// vite.config.js — hello vazio da fundação (F-01).
// Sobe com `task dev` para cumprir o aceite local ("frontend sobe, mesmo que
// com hello vazio"). Sem framework, sem negócio: só a página estática.
// Escuta SOMENTE em loopback; /api é proxied para o backend hello.
// Config real (Vue, TS, Orval) chega em F-07.
const backendPort = process.env.BACKEND_PORT || "8080";
const frontendPort = Number(process.env.FRONTEND_PORT || "5173");

export default {
  server: {
    host: "127.0.0.1",
    port: frontendPort,
    proxy: {
      "/api": `http://127.0.0.1:${backendPort}`,
    },
  },
};
