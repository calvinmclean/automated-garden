<script type="ts">
    import { onMount } from "svelte";
    import Gardens from "../components/Gardens.svelte";

    let gardens;

    onMount(async () => {
        await fetch(`http://localhost:8080/gardens`)
            .then((r) => r.json())
            .then((data) => {
                gardens = data.gardens;
            });
    });
</script>

<h1>Gardens</h1>
<Gardens {gardens} />
