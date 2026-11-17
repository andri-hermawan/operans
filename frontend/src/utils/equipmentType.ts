/** Status muatan (Loaded/Empty) hanya berlaku untuk Dump Truck. Unit tanpa tipe diperlakukan seperti Dump Truck. */
export const hasVessel = (type?: string | null) => !type || type === 'Dump Truck'
