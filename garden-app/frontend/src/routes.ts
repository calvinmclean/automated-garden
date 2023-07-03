// Components
import Gardens from './routes/Gardens.svelte'
import Garden from './routes/Garden.svelte'
import NotFound from './routes/NotFound.svelte'

// Export the route definition object
export default {
    // Exact path
    '/gardens': Gardens,

    // Using named parameters, with last being optional
    '/gardens/:garden_id': Garden,

    // Catch-all, must be last
    '*': NotFound,
}
