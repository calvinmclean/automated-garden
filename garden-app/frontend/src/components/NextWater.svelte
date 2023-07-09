<script lang="ts">
    import { Icon } from "sveltestrap";
    import type { NextWaterDetails } from "../lib/waterScheduleClient";

    export let nextWater: NextWaterDetails;
</script>

Next Water Time: {nextWater.time}
<Icon name="clock" /><br />
Next Water Duration: {nextWater.duration}<br />
Next Water Message: {nextWater.message}<br />
