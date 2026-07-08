<script>
  import {
    getIndicators,
    onIndicatorsUpdate,
    openActionCenter,
    openNetworkFlyout,
    openSoundFlyout,
    setVolume,
    toggleMute,
  } from './wails.js';
  import {
    Battery as BatteryIcon,
    BatteryCharging as BatteryChargingIcon,
    Wifi as WifiIcon,
    Cable as CableIcon,
    WifiOff as WifiOffIcon,
    VolumeX as VolumeXIcon,
    Volume as VolumeIcon,
    Volume1 as Volume1Icon,
    Volume2 as Volume2Icon,
  } from '@lucide/svelte';

  let { showBattery = true, showNetwork = true, showVolume = true } = $props();

  let battery = $state(null);
  let network = $state(null);
  let volume = $state(null);
  let showVolumeSlider = $state(false);

  function apply(payload) {
    if (payload?.battery) battery = payload.battery;
    if (payload?.network) network = payload.network;
    if (payload?.volume) volume = payload.volume;
  }

  getIndicators().then(apply).catch(() => {});
  onIndicatorsUpdate(apply);

  function onVolumeInput(event) {
    const value = Number(event.currentTarget.value) / 100;
    setVolume(value);
  }
</script>

<div class="flex items-center gap-1.5 text-[13px] h-full">
  {#if showNetwork && network}
    <button
      type="button"
      class="flex items-center justify-center p-1.5 border-0 bg-transparent text-inherit cursor-pointer rounded-md hover:bg-white/5 transition-colors duration-150"
      title={network.name ?? (network.connected ? 'Connected' : 'Disconnected')}
      onclick={() => openNetworkFlyout()}
    >
      {#if !network.connected}
        <WifiOffIcon class="w-4 h-4 text-white/40" />
      {:else if network.type === 'wifi'}
        <WifiIcon class="w-4 h-4" />
      {:else}
        <CableIcon class="w-4 h-4" />
      {/if}
    </button>
  {/if}

  {#if showBattery && battery?.present}
    <button
      type="button"
      class="flex items-center gap-1.5 px-2 py-1 border-0 bg-transparent text-inherit cursor-pointer rounded-md hover:bg-white/5 transition-colors duration-150"
      title={`${battery.percent}%${battery.charging ? ' (Charging)' : ''}`}
      onclick={() => openActionCenter()}
    >
      {#if battery.charging}
        <BatteryChargingIcon class="w-4 h-4 text-green-400" />
      {:else}
        <BatteryIcon class="w-4 h-4" />
      {/if}
      <span class="font-medium">{battery.percent}%</span>
    </button>
  {/if}

  {#if showVolume && volume}
    <div
      class="flex items-center gap-1.5 px-2 py-1 rounded-md hover:bg-white/5 transition-colors duration-150 relative"
      role="group"
      onmouseenter={() => (showVolumeSlider = true)}
      onmouseleave={() => (showVolumeSlider = false)}
    >
      <button
        type="button"
        class="border-0 bg-transparent text-inherit cursor-pointer p-0 flex items-center"
        title="Toggle mute (click), open sound settings (double-click)"
        onclick={() => toggleMute()}
        ondblclick={() => openSoundFlyout()}
      >
        {#if volume.muted}
          <VolumeXIcon class="w-4 h-4 text-white/40" />
        {:else if volume.level < 0.34}
          <VolumeIcon class="w-4 h-4" />
        {:else}
          <Volume2Icon class="w-4 h-4" />
        {/if}
      </button>
      {#if showVolumeSlider}
        <input
          class="w-16 h-1 bg-white/20 rounded-lg appearance-none cursor-pointer accent-[var(--pinky-accent)] focus:outline-none transition-all duration-150"
          type="range"
          min="0"
          max="100"
          value={Math.round((volume.level ?? 0) * 100)}
          oninput={onVolumeInput}
        />
      {/if}
    </div>
  {/if}
</div>
