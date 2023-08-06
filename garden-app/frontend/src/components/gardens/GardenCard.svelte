<script lang="ts">
    import {
        Badge,
        Button,
        ButtonDropdown,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        CardSubtitle,
        CardText,
        CardTitle,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
    } from "sveltestrap";
    import { fly } from "svelte/transition";

    import type { GardenResponse, getGarden } from "../../lib/gardenClient";
    import { lightAction, stopAction } from "../../lib/gardenClient";

    export let garden: GardenResponse;
    export let withLink = false;

    function toggleLight(event) {
        lightAction(garden.id, "");
    }

    function lightOn(event) {
        lightAction(garden.id, "ON");
    }

    function lightOff(event) {
        lightAction(garden.id, "OFF");
    }

    function stopWatering(event) {
        stopAction(garden.id);
    }
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

                {#if garden.temperature_humidity_data != null}
                    Temperature: {garden.temperature_humidity_data.temperature_celsius * 1.8 + 32}°F
                    <br />
                    Humidity: {garden.temperature_humidity_data.humidity_percentage}%
                    <br />
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
                        name={garden.next_light_action.state == "ON" ? "sunrise" : "sunset"}
                        style="color: {garden.next_light_action.state == 'ON' ? 'orange' : 'gray'}"
                    />
                    <br />
                {/if}

                {#if garden.end_date == null}
                    {#if garden.light_schedule != null}
                        <ButtonDropdown>
                            <DropdownToggle color="warning" caret>
                                <Icon name="sun" />
                            </DropdownToggle>
                            <DropdownMenu>
                                <DropdownItem on:click={toggleLight}>Toggle</DropdownItem>
                                <DropdownItem on:click={lightOn}>ON</DropdownItem>
                                <DropdownItem on:click={lightOff}>OFF</DropdownItem>
                            </DropdownMenu>
                        </ButtonDropdown>
                    {/if}

                    <Button color="danger" on:click={stopWatering}>
                        <Icon name="sign-stop" />
                    </Button>
                {/if}
            </CardText>
        </CardBody>
        <CardFooter>
            {#if garden.end_date != null}
                <Badge color={"danger"}>End Dated</Badge>
            {/if}

            {#if garden.health != null}
                <Icon name="wifi" style="color: {garden.health.status == 'UP' ? 'green' : 'red'}" />
            {/if}

            {#if garden.next_light_action != null}
                <Icon
                    name={garden.next_light_action.state == "ON" ? "sunrise" : "sunset"}
                    style="color: {garden.next_light_action.state == 'ON' ? 'orange' : 'gray'}"
                />
            {/if}

            <Icon name="grid" />{garden.num_zones}
        </CardFooter>
    </Card>
</div>
