<script lang="ts">
    import type { WaterScheduleResponse } from "../../lib/waterScheduleClient";
    import WaterScheduleCard from "./WaterScheduleCard.svelte";

    export let waterSchedule: WaterScheduleResponse;
    export let loadingWeatherData = false;
</script>

{#if waterSchedule}
    <WaterScheduleCard {waterSchedule} {loadingWeatherData} withLink={false} />
{/if}
