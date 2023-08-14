<script lang="ts">
    import { Accordion, AccordionItem, Spinner } from "sveltestrap";
    import ZoneCard from "./ZoneCard.svelte";
    import type { ZoneResponse } from "../../lib/zoneClient";
    import Zone from "./Zone.svelte";
    import { createZoneStore } from "../../store";

    export let gardenID: string;

    let zones: ZoneResponse[];
    let loadingWeatherData = true;

    let zoneStore = createZoneStore(gardenID);
    zoneStore.subscribe((zd) => {
        zones = zd.zones;
        loadingWeatherData = zd.loading;
    });

    const filterZones = (zones: ZoneResponse[], endDated: boolean) =>
        zones.filter((z) => (endDated ? z.end_date != null : z.end_date == null)).sort((a, b) => a.name.localeCompare(b.name));
</script>

{#if zones && zones.length > 1}
    {#each filterZones(zones, false) as zone (zone.id)}
        <ZoneCard {gardenID} {zone} {loadingWeatherData} />
    {/each}

    {#if filterZones(zones, true).length != 0}
        <Accordion flush>
            <AccordionItem header="End Dated Zones">
                {#each filterZones(zones, true) as zone (zone.id)}
                    <ZoneCard {gardenID} {zone} {loadingWeatherData} />
                {/each}
            </AccordionItem>
        </Accordion>
    {/if}
{:else if zones && zones.length == 1}
    <Zone {gardenID} zone={zones[0]} {loadingWeatherData} />
{:else}
    <Spinner color={"success"} type="border" />
{/if}
