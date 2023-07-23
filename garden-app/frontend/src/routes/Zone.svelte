<script lang="ts">
    import { onMount } from "svelte";
    import Zone from "../components/zones/Zone.svelte";
    import { getZone } from "../lib/zoneClient";
    import type { ZoneResponse, GetZoneParams } from "../lib/zoneClient";

    export let params: GetZoneParams;

    let zone: ZoneResponse;
    let loadingWeatherData = true;

    // quickly get zones
    onMount(async () => {
        await getZone(params.gardenID, params.zoneID, true)
            .then((response) => response.data)
            .then((data) => {
                zone = data;
            });
    });
    // then get with full details
    onMount(async () => {
        await getZone(params.gardenID, params.zoneID, false)
            .then((response) => response.data)
            .then((data) => {
                loadingWeatherData = false;
                zone = data;
            });
    });
</script>

{#if zone}
    <Zone gardenID={params.gardenID} {zone} {loadingWeatherData} />
{/if}
