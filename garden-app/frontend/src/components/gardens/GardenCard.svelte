<script lang="ts">
    import {
        Badge,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
        Icon,
    } from "sveltestrap";
    import { fly } from "svelte/transition";

    import type { GardenResponse } from "../../lib/gardenClient";

    export let garden: GardenResponse;
    export let withLink = false;
</script>

<div in:fly={{ x: 50, duration: 500 }} out:fly={{ x: -50, duration: 250 }}>
    <Card class=".col-lg-4" style="margin: 5%">
        {#if withLink}
            <a href="#/gardens/{garden.id}">
                <CardHeader>
                    <CardTitle>{garden.name}</CardTitle>
                </CardHeader>
            </a>
        {:else}
            <CardHeader>
                <CardTitle>{garden.name}</CardTitle>
            </CardHeader>
        {/if}
        <CardBody>
            <CardSubtitle>{garden.id}</CardSubtitle>
            <CardText>
                Topic prefix: {garden.topic_prefix}
                <Icon name="globe2" /><br />
                {#if garden.end_date != null}
                    End Dated: {garden.end_date}
                    <Icon name="clock-fill" style="color: red" /><br />
                {/if}

                {garden.num_zones} Zones <Icon name="grid" /><br />
                {garden.num_plants} Plants <Icon name="" /><br />

                {#if garden.health != null}
                    Health Status: {garden.health.status}<br />
                    Health Details: {garden.health.details}<br />
                {:else if garden.end_date == null}
                    No health details available<br />
                {/if}

                {#if garden.light_schedule != null}
                    Light Schedule Duration: {garden.light_schedule.duration}
                    <Icon name="hourglass-split" /><br />
                    Light Schedule Start: {garden.light_schedule.start_time}
                    <Icon name="clock" /><br />
                {/if}

                {#if garden.next_light_action != null}
                    Next Light Time: {garden.next_light_action.time}
                    <Icon name="clock" /><br />
                    Next Light State: {garden.next_light_action.state}
                    <Icon
                        name={garden.next_light_action.state == "ON"
                            ? "sunrise"
                            : "sunset"}
                        style="color: {garden.next_light_action.state == 'ON'
                            ? 'orange'
                            : 'gray'}"
                    />
                    <br />
                {/if}
            </CardText>
        </CardBody>
        <CardFooter>
            {#if garden.end_date != null}
                <Badge color={"danger"}>End Dated</Badge>
            {/if}

            {#if garden.health != null}
                <Icon
                    name="wifi"
                    style="color: {garden.health.status == 'UP'
                        ? 'green'
                        : 'red'}"
                />
            {/if}

            {#if garden.next_light_action != null}
                <Icon
                    name={garden.next_light_action.state == "ON"
                        ? "sunrise"
                        : "sunset"}
                    style="color: {garden.next_light_action.state == 'ON'
                        ? 'orange'
                        : 'gray'}"
                />
            {/if}

            <Icon name="grid" />{garden.num_zones}
        </CardFooter>
    </Card>
</div>
