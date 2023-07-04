import Gardens from './routes/Gardens.svelte'
import Garden from './routes/Garden.svelte'
import Zone from './routes/Zone.svelte'
import NotFound from './routes/NotFound.svelte'

export default {
    '/gardens': Gardens,
    '/gardens/:gardenID': Garden,
    '/gardens/:gardenID/zones/:zoneID': Zone,

    '*': NotFound,
}
