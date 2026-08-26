async function loadTasks() {
  const response = await fetch('/api/maintenance-tasks');
  const data = await response.json();
  document.querySelector('#orders').innerHTML = data.maintenance_tasks.map((task) =>
    `<li>${task.id} · ${task.array} · ${task.issue} · ${task.status}</li>`).join('');
}
document.querySelector('#refresh').addEventListener('click', loadTasks);
loadTasks();
