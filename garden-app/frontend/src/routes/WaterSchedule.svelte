<script lang="ts">
    import { onMount } from "svelte";
    import WaterSchedule from "../components/waterSchedules/WaterSchedule.svelte";
    import { getWaterSchedule, type WaterScheduleResponse, type GetWaterScheduleParams } from "../lib/waterScheduleClient";
    import { waterScheduleStore } from "../store";

    export let params: GetWaterScheduleParams;

    let waterSchedule: WaterScheduleResponse;

    waterScheduleStore.subscribe((value) => {
        waterSchedule = waterScheduleStore.getByID(value, params.waterScheduleID);
    });
</script>

<WaterSchedule {waterSchedule} />
