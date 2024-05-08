document.addEventListener('DOMContentLoaded', function () {
    const audioForm = document.getElementById('audioForm');
    const audioPlayerDiv = document.getElementById('audioPlayer');

    audioForm.addEventListener('submit', function (event) {
        event.preventDefault();
        const formData = new FormData(audioForm);
        const prompt = formData.get('prompt');

        const eventSource = new EventSource('/progress?prompt=' + encodeURIComponent(prompt));
        eventSource.onmessage = function (event) {
            const data = event.data;
            if (data.includes('audio:')) {
                var audio = data.split(':')[1].trim();
                audioPlayerDiv.innerHTML = "<audio controls><source src=\"data:audio/mpeg;base64," + audio + "\" type=\"audio/mpeg\"></audio>"
            } else {
               audioPlayerDiv.textContent = data;
            }
        };

        eventSource.onerror = function () {
            eventSource.close();
        };
    });
});

