function fetchTasks() {
    fetch('/tasks')
        .then(response => response.json())
        .then(data => {
            const taskList = document.getElementById('taskList');
            taskList.innerHTML = '';
            data.forEach(task => {
                const li = document.createElement('li');
                li.innerHTML = `
                    <input type="checkbox" ${task.completed ? 'checked' : ''} onchange="updateTasks()">
                    <span class="${task.completed ? 'completed' : ''}">${task.text}</span>
                    <button class="edit" onclick="editTask(this)">Edit</button>
                    <button class="remove" onclick="removeTask(this)">Hapus</button>
                `;
                taskList.appendChild(li);
            });
        });
}

function addTask() {
    const taskInput = document.getElementById('taskInput');
    const taskValue = taskInput.value.trim();

    if (taskValue === '') {
        alert('Tugas tidak boleh kosong!');
        return;
    }

    fetch('/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: taskValue, completed: false }),
    }).then(() => {
        taskInput.value = '';
        fetchTasks();
    });
}

function removeTask(button) {
    const taskList = document.getElementById('taskList');
    const taskItem = button.parentElement;
    const taskText = taskItem.querySelector('span').textContent;

    // Send DELETE request to server
    fetch('/tasks', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: taskText })
    }).then(() => {
        taskList.removeChild(taskItem);
    });
}

function editTask(button) {
    const taskItem = button.parentElement;
    const taskText = taskItem.querySelector('span');
    const currentText = taskText.innerText;

    // Replace span with an input field for editing
    const inputField = document.createElement('input');
    inputField.type = 'text';
    inputField.value = currentText;

    taskItem.replaceChild(inputField, taskText);
    button.textContent = 'Simpan';
    button.onclick = function () {
        const newText = inputField.value.trim();
        taskText.innerText = newText;
        taskItem.replaceChild(taskText, inputField);
        button.textContent = 'Edit';

        // Send PUT request to update task
        fetch('/tasks', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text: newText })
        }).then(() => {
            button.onclick = () => editTask(button);
        });
    };
}

window.onload = fetchTasks;
