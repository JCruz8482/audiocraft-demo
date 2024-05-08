const ERROR_DISPLAY_MESSAGE = "uh oh, there was an error"

document.addEventListener('DOMContentLoaded', function () {
    const audioForm = document.getElementById('audioForm');
    const audioPlayerDiv = document.getElementById('audioPlayer');

    audioForm.addEventListener('submit', function (event) {
        event.preventDefault();
        const formData = new FormData(audioForm);
        const prompt = formData.get('prompt');

        const eventSource = new EventSource('/generateAudio?prompt=' + encodeURIComponent(prompt));
        eventSource.onmessage = function (event) {
            if (event.data) {
                const data = event.data;
                if (data.includes('audio:')) {
                    var audio = data.split(':')[1].trim();
                    audioPlayerDiv.innerHTML = "<audio controls><source src=\"data:audio/mpeg;base64," + audio + "\" type=\"audio/mpeg\"></audio>"
                    eventSource.close()
                } else {
                    audioPlayerDiv.textContent = data;
                }
            } else {
                audioPlayerDiv.textContent = ERROR_DISPLAY_MESSAGE;
                eventSource.close()
            }
        };

        eventSource.onerror = function () {
            audioPlayerDiv.textContent = ERROR_DISPLAY_MESSAGE;
            eventSource.close();
        };
    });
});

