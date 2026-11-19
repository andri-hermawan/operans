// Tangkap satu elemen halaman menjadi gambar PNG tanpa pustaka tambahan:
// salin elemen + gaya hasil hitung (computed style) ke dalam SVG <foreignObject>, lalu gambar ke canvas.
// Cocok untuk kartu sederhana (teks, ikon SVG, warna). Elemen bertanda data-capture-ignore tidak ikut.

const XHTML = 'http://www.w3.org/1999/xhtml'
const SVG_NS = 'http://www.w3.org/2000/svg'

/** Salin gaya hasil hitung dari pohon asli ke pohon salinan (urutan simpul sama). */
function inlineStyles(src: Element, dst: Element) {
  const cs = getComputedStyle(src)
  const target = dst as HTMLElement
  for (let i = 0; i < cs.length; i++) {
    const name = cs[i]
    target.style.setProperty(name, cs.getPropertyValue(name), cs.getPropertyPriority(name))
  }
  target.style.setProperty('animation', 'none')
  target.style.setProperty('transition', 'none')
  for (let i = 0; i < src.children.length; i++) inlineStyles(src.children[i], dst.children[i])
}

/** Gambar <img> harus berupa data URL agar ikut tergambar di dalam SVG. Gagal memuat = dibiarkan. */
async function inlineImages(root: HTMLElement) {
  await Promise.all(
    [...root.querySelectorAll('img')].map(async (img) => {
      try {
        const blob = await (await fetch(img.src)).blob()
        const data = await new Promise<string>((resolve, reject) => {
          const r = new FileReader()
          r.onload = () => resolve(String(r.result))
          r.onerror = () => reject(r.error)
          r.readAsDataURL(blob)
        })
        img.src = data
      } catch {
        /* biarkan */
      }
    }),
  )
}

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('Gagal membuat gambar'))
    img.src = url
  })
}

export async function captureNodeToPng(node: HTMLElement, opts: { scale?: number; background?: string } = {}): Promise<Blob> {
  const { scale = 2, background = '#ffffff' } = opts
  const rect = node.getBoundingClientRect()
  const width = Math.ceil(rect.width)
  const height = Math.ceil(rect.height)

  const clone = node.cloneNode(true) as HTMLElement
  inlineStyles(node, clone)
  clone.querySelectorAll('[data-capture-ignore]').forEach((el) => el.remove())
  await inlineImages(clone)
  clone.style.setProperty('margin', '0')
  clone.style.setProperty('width', `${width}px`)
  clone.style.setProperty('height', `${height}px`)
  clone.style.setProperty('box-sizing', 'border-box')
  clone.setAttribute('xmlns', XHTML)

  const svg = document.createElementNS(SVG_NS, 'svg')
  svg.setAttribute('xmlns', SVG_NS)
  svg.setAttribute('width', String(width))
  svg.setAttribute('height', String(height))
  const fo = document.createElementNS(SVG_NS, 'foreignObject')
  fo.setAttribute('width', '100%')
  fo.setAttribute('height', '100%')
  fo.appendChild(clone)
  svg.appendChild(fo)

  const url = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(new XMLSerializer().serializeToString(svg))}`
  const img = await loadImage(url)

  const canvas = document.createElement('canvas')
  canvas.width = width * scale
  canvas.height = height * scale
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Canvas tidak tersedia')
  ctx.fillStyle = background
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  ctx.scale(scale, scale)
  ctx.drawImage(img, 0, 0, width, height)

  return new Promise((resolve, reject) => canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('Gagal membuat PNG'))), 'image/png'))
}

/** Simpan Blob sebagai berkas unduhan. */
export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
