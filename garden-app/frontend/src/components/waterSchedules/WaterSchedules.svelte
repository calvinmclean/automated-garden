<script lang="ts">
    import { Accordion, AccordionItem } from "sveltestrap";
    import { onMount } from "svelte";

    import WaterScheduleCard from "./WaterScheduleCard.svelte";
    import { getWaterSchedules, type WaterScheduleResponse } from "../../lib/waterScheduleClient";
    import { waterScheduleStore } from "../../store";

    let waterSchedules: WaterScheduleResponse[];
    let loadingWeatherData = true;

    waterScheduleStore.subscribe((wsData) => {
        waterSchedules = wsData.waterSchedules;
        loadingWeatherData = wsData.loading;
    });

    const filterWaterSchedules = (waterSchedules: WaterScheduleResponse[], endDated: boolean) =>
        waterSchedules.filter((ws) => (endDated ? ws.end_date != null : ws.end_date == null)).sort((a, b) => a.name.localeCompare(b.name));
</script>

{#if waterSchedules}
    {#each filterWaterSchedules(waterSchedules, false) as waterSchedule (waterSchedule.id)}
        <WaterScheduleCard {waterSchedule} {loadingWeatherData} withLink={true} />
    {/each}

    {#if filterWaterSchedules(waterSchedules, true).length != 0}
        <Accordion flush>
            <AccordionItem header="End Dated Water Schedules">
                {#each filterWaterSchedules(waterSchedules, true) as waterSchedule (waterSchedule.id)}
                    <WaterScheduleCard {waterSchedule} {loadingWeatherData} withLink={true} />
                {/each}
            </AccordionItem>
        </Accordion>
    {/if}
{/if}
