import { describe, expect, it } from 'vitest'

import { withoutOutliers } from './map'

describe('withoutOutliers', () => {
  it('membuang titik yang jauh dari kelompok unit', () => {
    const cluster: [number, number][] = [[-3.48, 103.87], [-3.49, 103.86], [-3.485, 103.875], [-3.47, 103.88]]
    expect(withoutOutliers([...cluster, [0, 0]])).toEqual(cluster)
  })
  it('titik sedikit dibiarkan apa adanya', () => {
    expect(withoutOutliers([[0, 0], [-3, 103]])).toHaveLength(2)
  })
})
