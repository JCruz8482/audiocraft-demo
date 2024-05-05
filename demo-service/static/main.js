document.addEventListener('DOMContentLoaded', function () {
    const audioForm = document.getElementById('audioForm');
    const progressDiv = document.getElementById('progress');
    const resultDiv = document.getElementById('result');

    audioForm.addEventListener('submit', function (event) {
        event.preventDefault();
        const formData = new FormData(audioForm);
        const prompt = formData.get('prompt');

        const eventSource = new EventSource('/progress?prompt=' + encodeURIComponent(prompt));

        eventSource.onmessage = function (event) {
            const data = event.data;
            progressDiv.textContent = "Recieved data: " + data;
            console.log('Received progress:', data);
        };

        eventSource.onerror = function () {
            console.error('SSE connection error');
            eventSource.close();
        };
    });
});
