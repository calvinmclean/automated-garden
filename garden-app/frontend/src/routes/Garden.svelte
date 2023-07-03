<script>
    import { onMount } from "svelte";
    import Garden from "../components/Garden.svelte";

    export let params;
    let garden;

    onMount(async () => {
        await fetch(`http://localhost:8080/gardens/` + params.garden_id)
            .then((r) => r.json())
            .then((data) => {
                garden = data;
            });
    });
</script>

<Garden {garden} />
