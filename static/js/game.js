document.addEventListener('DOMContentLoaded', () => {
    const timerDisplay = document.getElementById('timer');
    const slides = document.querySelectorAll('.slide');
    const loader = document.getElementById('loader');
    const slideContainer = document.getElementById('slide-container');
    const nextParticipantForm = document.getElementById('next-participant-form');
    let currentSlide = 0;
    let timeLeft = 60;

    const participantName = window.location.pathname.split('/').pop();

    // Show more informative loading message
    const loadingMessages = [
        "Generating your presentation...",
        "Creating absurd business ideas...",
        "Crafting ridiculous images...",
        "Preparing comedy gold...",
        "Almost ready for your moment of glory..."
    ];
    
    let messageIndex = 0;
    const messageInterval = setInterval(() => {
        if (messageIndex < loadingMessages.length - 1) {
            messageIndex++;
            loader.querySelector('p').textContent = loadingMessages[messageIndex];
        }
    }, 3000);

    fetch(`/api/game-data/${participantName}`)
        .then(response => response.json())
        .then(data => {
            clearInterval(messageInterval);
            
            document.getElementById('business-name').textContent = data.businessName;
            document.getElementById('slogan').textContent = data.slogan;
            document.getElementById('image1').src = data.image1;
            document.getElementById('image2').src = data.image2;
            document.getElementById('clapping-gif').src = data.clappingGif;

            loader.style.display = 'none';
            slideContainer.style.display = 'block';
            slides[0].style.display = 'flex';

            startTimer();
        })
        .catch(error => {
            clearInterval(messageInterval);
            console.error('Error fetching game data:', error);
            loader.innerHTML = '<p>Failed to load game data. Please try again.</p>';
        });


    const advanceSlide = () => {
        if (currentSlide < slides.length - 1) {
            slides[currentSlide].style.display = 'none';
            currentSlide++;
            slides[currentSlide].style.display = 'flex';
        }
    };

    const startTimer = () => {
        const timerInterval = setInterval(() => {
            timeLeft--;
            const minutes = Math.floor(timeLeft / 60);
            const seconds = timeLeft % 60;
            timerDisplay.textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;

            if (timeLeft % 15 === 0) {
                advanceSlide();
            }

            if (timeLeft <= 0) {
                clearInterval(timerInterval);
                timerDisplay.textContent = "Time's Up!";
                nextParticipantForm.style.display = 'block';
            }
        }, 1000);
    }
}); 