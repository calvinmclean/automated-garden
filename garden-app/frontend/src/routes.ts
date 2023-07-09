import Gardens from './routes/Gardens.svelte'
import Garden from './routes/Garden.svelte'
import Zone from './routes/Zone.svelte'
import NotFound from './routes/NotFound.svelte'
import WaterSchedules from './routes/WaterSchedules.svelte'
import WaterSchedule from './routes/WaterSchedule.svelte'

export default {
    '/gardens': Gardens,
    '/gardens/:gardenID': Garden,
    '/gardens/:gardenID/zones/:zoneID': Zone,
    '/water_schedules': WaterSchedules,
    '/water_schedules/:waterScheduleID': WaterSchedule,

    '*': NotFound,
}
