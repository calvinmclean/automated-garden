<script lang="ts">
    import {
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
    } from "sveltestrap";
    import type { components } from "../types/garden-app-openapi";

    type GardenResponse = components["schemas"]["GardenResponse"];

    export let garden: GardenResponse;
</script>

<Card class="mb-3">
    <CardHeader>
        <CardTitle>
            <a href="#/gardens/{garden.id}">{garden.name}</a>
        </CardTitle>
    </CardHeader>
    <CardBody>
        <CardSubtitle>{garden.id}</CardSubtitle>
        <CardText>
            <li>{garden.topic_prefix}</li>
        </CardText>
    </CardBody>
    <CardFooter>{garden.created_at}</CardFooter>
</Card>
