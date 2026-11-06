// Alamat dasar aplikasi (mis. "/operans"), diatur lewat `base` di vite.config.ts.
export const BASE_PATH = import.meta.env.BASE_URL.replace(/\/$/, '')

/** Awali path internal dengan alamat dasar, mis. appUrl('/print/x') → "/operans/print/x". */
export const appUrl = (path: string) => `${BASE_PATH}${path}`
