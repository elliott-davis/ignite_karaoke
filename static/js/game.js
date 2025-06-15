document.addEventListener('DOMContentLoaded', () => {
    const timerDisplay = document.getElementById('timer');
    const slides = document.querySelectorAll('.slide');
    let currentSlide = 0;
    let timeLeft = 60;

    const advanceSlide = () => {
        if (currentSlide < slides.length - 1) {
            slides[currentSlide].style.display = 'none';
            currentSlide++;
            slides[currentSlide].style.display = 'flex';
        }
    };

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
}); 