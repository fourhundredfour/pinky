<script>
  import Tasks from './lib/Tasks.svelte';
  import Clock from './lib/Clock.svelte';
  import Indicators from './lib/Indicators.svelte';
  import { getConfig, onConfigUpdate } from './lib/wails.js';

  let config = $state(null);

  const alignmentJustify = {
    left: 'flex-start',
    center: 'center',
    right: 'flex-end',
  };

  let taskZoneJustify = $derived(alignmentJustify[config?.alignment ?? 'center'] ?? 'center');
  let shapeRadius = $derived(config?.shape === 'rounded' ? '14px' : '0px');

  function applyTheme(cfg) {
    config = cfg;
    const root = document.documentElement;
    root.style.setProperty('--pinky-accent', cfg.accent_color ?? '#3aa0ff');
    root.style.setProperty('--pinky-bg', cfg.background_color ?? '#101014');
    root.style.setProperty('--pinky-bg-opacity', String(cfg.background_opacity ?? 0.85));
    root.classList.toggle('pinky-monochrome', Boolean(cfg.monochrome_icons));
  }

  // The backend only pushes config:update on save/reload, so fetch the
  // config once up front for the very first render.
  getConfig().then(applyTheme).catch(() => {});
  onConfigUpdate(applyTheme);
</script>

<div
  class="flex items-stretch w-screen h-screen text-[#f5f5f7] overflow-hidden box-border"
  style="border-radius: {shapeRadius}; background-color: color-mix(in srgb, var(--pinky-bg, #101014) calc(var(--pinky-bg-opacity, 0.85) * 100%), transparent);"
>
  <div class="flex items-center min-w-0 flex-1 overflow-hidden px-2" style:justify-content={taskZoneJustify}>
    {#if config?.show_tasks ?? true}
      <Tasks />
    {/if}
  </div>
  <div class="flex items-center min-w-0 flex-none gap-3.5 px-3">
    {#if config?.show_clock ?? true}
      <Clock format={config?.clock_format ?? '15:04'} />
    {/if}
    <Indicators
      showBattery={config?.show_battery ?? true}
      showNetwork={config?.show_network ?? true}
      showVolume={config?.show_volume ?? true}
    />
  </div>
</div>
