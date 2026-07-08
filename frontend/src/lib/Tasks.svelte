<script>
  import { onTasksUpdate, focusTask, closeTask, listTasks } from './wails.js';

  let tasks = $state([]);

  listTasks().then((list) => {
    tasks = list ?? [];
  }).catch(() => {});

  onTasksUpdate((list) => {
    tasks = list ?? [];
  });

  function handleClick(id) {
    focusTask(id);
  }

  function handleMiddleClick(event, id) {
    if (event.button !== 1) return;
    event.preventDefault();
    closeTask(id);
  }
</script>

<div class="flex items-center gap-1 h-full overflow-hidden">
  {#each tasks as task (task.id)}
    <button
      type="button"
      class="flex items-center justify-center w-10 h-10 p-0 border-0 rounded-lg bg-transparent cursor-pointer transition-all duration-150 relative hover:bg-white/10 active:scale-95 will-change-transform transform-gpu"
      class:bg-white-15={task.active}
      class:opacity-50={task.minimized}
      title={task.title}
      onclick={() => handleClick(task.id)}
      onmouseup={(event) => handleMiddleClick(event, task.id)}
    >
      {#if task.icon}
        <img class="task-icon w-5 h-5 object-contain" src={task.icon} alt="" />
      {:else}
        <span class="w-2.5 h-2.5 rounded-full bg-current opacity-50" aria-hidden="true"></span>
      {/if}

      {#if task.active}
        <!-- Active indicator dot/bar at the bottom -->
        <span class="absolute bottom-1.5 left-1/2 -translate-x-1/2 w-3 h-0.5 rounded-full bg-[var(--pinky-accent)]"></span>
      {:else if !task.minimized}
        <!-- Running but inactive indicator dot -->
        <span class="absolute bottom-1.5 left-1/2 -translate-x-1/2 w-1 h-1 rounded-full bg-white/40"></span>
      {/if}
    </button>
  {/each}
</div>

<style>
  /* Custom utility for active background opacity */
  .bg-white-15 {
    background-color: rgba(255, 255, 255, 0.15);
  }
</style>
