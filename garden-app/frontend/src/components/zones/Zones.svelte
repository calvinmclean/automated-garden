<script lang="ts">
    import { Spinner } from "sveltestrap";
    import { onMount } from "svelte";
    import { getZones } from "../../lib/zoneClient";
    import ZoneCard from "./ZoneCard.svelte";
    import type { ZoneResponse } from "../../lib/zoneClient";
    import Zone from "./Zone.svelte";

    export let gardenID: string;

    let zones: ZoneResponse[];
    let loadingWeatherData = true;

    // quickly get zones
    onMount(async () => {
        await getZones(gardenID, true, true)
            .then((response) => response.data)
            .then((data) => {
                zones = data.zones;
            });
    });
    // then get with full details
    onMount(async () => {
        await getZones(gardenID, true, false)
            .then((response) => response.data)
            .then((data) => {
                loadingWeatherData = false;
                zones = data.zones;
            });
    });
</script>

{#if zones && zones.length > 1}
    {#each zones as zone}
        <ZoneCard {zone} withLink={true} {loadingWeatherData} />
    {/each}
{:else if zones && zones.length == 1}
    <Zone {gardenID} zone={zones[0]} {loadingWeatherData} />
{:else}
    <Spinner color={"success"} type="border" />
{/if}
