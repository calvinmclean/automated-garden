<script lang="ts">
    import { onMount } from "svelte";
    import Zone from "../components/zones/Zone.svelte";
    import { getZone } from "../lib/zoneClient";
    import type { ZoneResponse, GetZoneParams } from "../lib/zoneClient";

    export let params: GetZoneParams;

    let zone: ZoneResponse;

    onMount(async () => {
        await getZone(params.gardenID, params.zoneID)
            .then((response) => response.data)
            .then((data) => {
                zone = data;
            });
    });
</script>

<Zone gardenID={params.gardenID} {zone} />
