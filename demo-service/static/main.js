const ERROR_DISPLAY_MESSAGE = "uh oh, there was an error"

document.addEventListener('DOMContentLoaded', function () {
    const audioForm = document.getElementById('audioForm');
    const audioPlayerDiv = document.getElementById('audioPlayer');
    
    
    audioForm.addEventListener('submit', async function (event) {
        event.preventDefault();
        const formData = new FormData(audioForm);
        const prompt = formData.get('prompt');

        fetch('/generateAudio', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ prompt: prompt })
        })
        .then(response => response.json())
        .then(data => {
            if (!data.id) {
                audioPlayerDiv.textContent = ERROR_DISPLAY_MESSAGE;
            }

            const eventSource = new EventSource(`/generateAudio/${data.id}`);

            eventSource.onmessage = function(event) {
                console.log(event)
                const eventData = JSON.parse(event.data);

                if (eventData.status === "Processing") {
                    audioPlayerDiv.innerText = eventData.status;
                } else if (eventData.status === "Done") {
                    audioPlayerDiv.innerHTML = `<audio controls src=${eventData.data}></audio>`;
                    eventSource.close();
                }
            };

            eventSource.onerror = function() {
                console.error('EventSource failed.');
                eventSource.close();
                audioPlayerDiv.textContent = ERROR_DISPLAY_MESSAGE;
            };
        })
        .catch(error => {
            console.error('Fetch error:', error);
            audioPlayerDiv.textContent = ERROR_DISPLAY_MESSAGE;
        });
    });
});
