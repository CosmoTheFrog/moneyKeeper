import { describe, expect, it } from 'vitest'
import { visibleCurrencies } from './currencyChoice'

const all = ['GEL', 'EUR', 'USD', 'RUB', 'BTC', 'ETH']
const plain = (c: string) => c !== 'BTC' && c !== 'ETH'

describe('visibleCurrencies', () => {
  it.each([
    ['no choice yet: only plain currencies', null, [], ['GEL', 'EUR', 'USD', 'RUB']],
    ['chosen list wins', ['GEL', 'ETH'], [], ['GEL', 'ETH']],
    ['base and account currencies always stay', ['EUR'], ['GEL', 'BTC'], ['GEL', 'EUR', 'BTC']],
    ['empty choice keeps only the locked ones', [], ['GEL'], ['GEL']],
  ])('%s', (_name, enabled, always, want) => {
    expect(visibleCurrencies(all, enabled, always, plain)).toEqual(want)
  })
})
