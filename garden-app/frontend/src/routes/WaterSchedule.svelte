<script lang="ts">
    import { onMount } from "svelte";
    import WaterSchedule from "../components/waterSchedules/WaterSchedule.svelte";
    import { getWaterSchedule, type WaterScheduleResponse, type GetWaterScheduleParams } from "../lib/waterScheduleClient";

    export let params: GetWaterScheduleParams;

    let waterSchedule: WaterScheduleResponse;

    onMount(async () => {
        await getWaterSchedule(params.waterScheduleID, false)
            .then((response) => response.data)
            .then((data) => {
                waterSchedule = data;
            });
    });
</script>

<WaterSchedule {waterSchedule} />
