document.addEventListener('DOMContentLoaded', () => {
    const timerDisplay = document.getElementById('timer');
    const slides = document.querySelectorAll('.slide');
    const loader = document.getElementById('loader');
    const slideContainer = document.getElementById('slide-container');
    let currentSlide = 0;
    let timeLeft = 60;

    const participantName = window.location.pathname.split('/').pop();

    fetch(`/api/game-data/${participantName}`)
        .then(response => response.json())
        .then(data => {
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
        const timer = setInterval(() => {
            timeLeft--;
            const minutes = Math.floor(timeLeft / 60);
            const seconds = timeLeft % 60;
            timerDisplay.textContent = `${minutes}:${seconds < 10 ? '0' : ''}${seconds}`;

            if (timeLeft % 15 === 0 && timeLeft > 0) {
                advanceSlide();
            }

            if (timeLeft <= 0) {
                clearInterval(timer);
                // Show the final slide
                slides[currentSlide].style.display = 'none';
                document.getElementById('slide5').style.display = 'flex';
            }
        }, 1000);
    }
}); 