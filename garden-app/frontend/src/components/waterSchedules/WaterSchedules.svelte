<script lang="ts">
    import { Accordion, AccordionItem } from "sveltestrap";
    import { onMount } from "svelte";

    import WaterScheduleCard from "./WaterScheduleCard.svelte";
    import { getWaterSchedules } from "../../lib/waterScheduleClient";
    import type { WaterScheduleResponse } from "../../lib/waterScheduleClient";

    let waterSchedules: WaterScheduleResponse[];
    let loadingWeatherData = true;

    // quickly get zones
    onMount(async () => {
        await getWaterSchedules(true, true)
            .then((response) => response.data)
            .then((data) => {
                waterSchedules = data.water_schedules;
            });
    });
    // then get with full details
    onMount(async () => {
        await getWaterSchedules(true, false)
            .then((response) => response.data)
            .then((data) => {
                loadingWeatherData = false;
                waterSchedules = data.water_schedules;
            });
    });

    const filterWaterSchedules = (
        waterSchedules: WaterScheduleResponse[],
        endDated: boolean
    ) =>
        waterSchedules
            .filter((ws) =>
                endDated ? ws.end_date != null : ws.end_date == null
            )
            .sort((a, b) => a.name.localeCompare(b.name));
</script>

{#if waterSchedules}
    {#each filterWaterSchedules(waterSchedules, false) as waterSchedule (waterSchedule.id)}
        <WaterScheduleCard
            {waterSchedule}
            {loadingWeatherData}
            withLink={true}
        />
    {/each}

    {#if filterWaterSchedules(waterSchedules, true).length != 0}
        <Accordion flush>
            <AccordionItem header="End Dated Water Schedules">
                {#each filterWaterSchedules(waterSchedules, true) as waterSchedule (waterSchedule.id)}
                    <WaterScheduleCard
                        {waterSchedule}
                        {loadingWeatherData}
                        withLink={true}
                    />
                {/each}
            </AccordionItem>
        </Accordion>
    {/if}
{/if}
