<script>
  import { onClockTick } from './wails.js';

  let { format = '15:04' } = $props();

  let timeText = $state('--:--');
  let dateText = $state('');

  onClockTick((payload) => {
    timeText = payload?.time ?? timeText;
    dateText = payload?.date ?? dateText;
  });
</script>

<div class="flex items-center justify-center px-2.5 text-[13px] font-semibold tracking-wide whitespace-nowrap cursor-default hover:bg-white/5 rounded-md py-1 transition-colors duration-150" title={dateText}>
  <span>{timeText}</span>
</div>
