// hooks/post-rotate.mjs — USER HOOK, bisa diedit dari dashboard.
// Dijalankan setiap autorotate / rotate sukses.
// Env yang tersedia: OLD_IP, NEW_IP, TIMESTAMP
// Cara run manual: OLD_IP=1.1.1.1 NEW_IP=2.2.2.2 bun hooks/post-rotate.mjs
const oldIP = process.env.OLD_IP ?? "-";
const newIP = process.env.NEW_IP ?? "-";
const ts = process.env.TIMESTAMP ?? new Date().toISOString();

console.log(`[post-rotate] ${oldIP} -> ${newIP} @ ${ts}`);

// === TULIS KODE KAMU DI BAWAH INI (dari dashboard juga bisa) ===
// Contoh: kirim webhook
// if (process.env.WEBHOOK_URL) {
//   await fetch(process.env.WEBHOOK_URL, {
//     method: "POST",
//     headers: { "content-type": "application/json" },
//     body: JSON.stringify({ oldIP, newIP, ts }),
//   });
// }
