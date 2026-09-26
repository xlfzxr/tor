// hooks/pre-rotate.mjs — USER HOOK, jalan SEBELUM NEWNYM.
// Env: OLD_IP, TIMESTAMP. Exit non-zero = batalkan rotate.
const oldIP = process.env.OLD_IP ?? "-";
console.log(`[pre-rotate] oldIP=${oldIP}`);

// Contoh guard: jangan rotate kalau IP masih muda (< 30s)
// (logika penuh ada di tor-autorotate, ini hanya contoh hook)
