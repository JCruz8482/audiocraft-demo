document.addEventListener('DOMContentLoaded', function () {
    const audioForm = document.getElementById('audioForm');
    const progressDiv = document.getElementById('progress');
    const resultDiv = document.getElementById('result');
    const audioPlayerDiv = document.getElementById('audioPlayer');

    audioForm.addEventListener('submit', function (event) {
        event.preventDefault();
        const formData = new FormData(audioForm);
        const prompt = formData.get('prompt');

        const eventSource = new EventSource('/progress?prompt=' + encodeURIComponent(prompt));

        eventSource.onmessage = function (event) {
            const data = event.data;
            if (data.includes('audio:')) {
                console.log("audio included")
                console.log(data)
                var parts = data.split(':');
                var audio = parts[1].trim();
                console.log("audio portion")
                console.log(audio)
                audioPlayerDiv.innerHTML = "<audio controls><source src=\"data:audio/mpeg;base64," + audio + "\" type=\"audio/mpeg\"></audio>"
            } else {
                progressDiv.textContent = "Recieved data: " + data;
                console.log('Received data:', data);
            }
        };

        eventSource.onerror = function () {
            console.error('SSE connection error');
            eventSource.close();
        };
    });
});
