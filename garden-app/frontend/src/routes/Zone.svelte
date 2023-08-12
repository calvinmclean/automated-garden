<script lang="ts">
    import { onMount } from "svelte";
    import Zone from "../components/zones/Zone.svelte";
    import { type ZoneResponse, type GetZoneParams } from "../lib/zoneClient";
    import { zoneStore } from "../store";

    export let params: GetZoneParams;

    let zone: ZoneResponse;
    let loadingWeatherData = true;

    zoneStore.init(params.gardenID);
    zoneStore.subscribe((value) => {
        loadingWeatherData = value.loading;
        zone = zoneStore.getByID(value, params.zoneID);
    });
</script>

{#if zone}
    <Zone gardenID={params.gardenID} {zone} {loadingWeatherData} />
{/if}
