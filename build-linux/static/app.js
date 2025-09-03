// Получение параметров из URL
const urlParams = new URLSearchParams(window.location.search);
const scannerMode = urlParams.get("mode") || "default";
const salt = urlParams.get("salt") || "salt";
const getParamEventId = urlParams.get("event_id") ? urlParams.get("event_id") : "default";
const getParamManagerName = urlParams.get("manager_name") ? urlParams.get("manager_name") : null;

// Установка заголовка
document.getElementById("title").textContent = "Регистрация пользователя на событии " + getParamEventId;

// Глобальные переменные
let currentUserId = null;
let POPUP_SHOWED = false;
let scanner = null;

// Настройка размеров видео контейнера
const win = window;
const doc = document;
const docElem = doc.documentElement;
const body = doc.getElementsByTagName('body')[0];
const x = win.innerWidth || docElem.clientWidth || body.clientWidth;
const y = win.innerHeight || docElem.clientHeight || body.clientHeight;

const videoContainer = document.getElementById("video-container");
videoContainer.style.width = x + "px";
videoContainer.style.height = y + "px";

// Элементы интерфейса
const popup = document.getElementById("popup");
const noBtn = document.getElementById("no-btn");
const yesBtn = document.getElementById("yes-btn");
const popupOneButton = document.getElementById("popup-one-button");
const popupOneButtonBtn = document.getElementById("popup-one-button-btn");
const overlay = document.getElementById("overlay");

// Функция обработки результата сканирования
function setResult(result) {
    if (!POPUP_SHOWED) {
        console.log(result.data);

        let regexp = /\/(user)\/([a-z0-9]+)/g;
        if (scannerMode === "secure") {
            regexp = /\/([a-z0-9]{2})\/(.+)/g;
        }

        console.log(regexp);

        let res = "";
        for (const match of result.data.matchAll(regexp)) {
            res = match[2];
            if (scannerMode === "secure") {
                const hash = CryptoJS.MD5(res + salt);
                console.log(res + salt);
                console.log(hash.toString());
                const hashShort = hash.toString().slice(0, 2);
                console.log(match);
                if (hashShort !== match[1]) {
                    res = "";
                }
            }
        }

        if (res === "") {
            currentUserId = null;
            showPopupOneButton("Некорректный QR", "Продолжить сканирование", true);
        } else {
            currentUserId = res;
            showResultPopup(res);
        }
    }
}

// Функция показа модального окна подтверждения
function showResultPopup(result) {
    POPUP_SHOWED = true;
    popup.style.display = "block";
    overlay.classList.add("show");
    document.getElementById("popup-header").innerHTML = 
        "Зарегистрировать пользователя с id<br/>" + result + "<br/>на событии " + getParamEventId + "?";
}

// Функция показа модального окна с одной кнопкой
function showPopupOneButton(headerText, buttonText, isError = false) {
    POPUP_SHOWED = true;
    popup.style.display = "none";
    overlay.classList.add("show");

    document.getElementById("popup-one-button-header").innerHTML = headerText;
    document.getElementById("popup-one-button-btn").textContent = buttonText;

    if (isError) {
        popupOneButton.className = "popup error-border";
    } else {
        popupOneButton.className = "popup success-border";
    }
    popupOneButton.style.display = "block";
}

// Обработчики событий для кнопок
noBtn.addEventListener("click", () => {
    POPUP_SHOWED = false;
    popup.style.display = "none";
    overlay.classList.remove("show");
});

yesBtn.addEventListener("click", () => {
    POPUP_SHOWED = false;
    popup.style.display = "none";

    const json = {
        user_id: currentUserId,
        event_id: getParamEventId,
        manager_name: getParamManagerName
    };

    // Отправка данных на сервер
    fetch("/scannerPostData", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(json)
    })
    .then(response => response.json())
    .then(data => {
        if (data.message === "ok") {
            showPopupOneButton("Пользователь успешно зарегистрирован", "Продолжить сканирование");
        } else if (data.message.includes("duplicate")) {
            showPopupOneButton("<p style='color: #ef8080;'>Ошибка!</p>Повторная регистрация пользователя!", "Продолжить сканирование", true);
        } else {
            showPopupOneButton("Техническая ошибка при добавлении: " + data.message, "Продолжить сканирование", true);
            console.log(data);
        }
    })
    .catch(error => {
        showPopupOneButton("Техническая ошибка при отправке запроса: " + error.toString(), "Продолжить сканирование", true);
        console.log(error);
    });

    overlay.classList.add("show");
});

popupOneButtonBtn.addEventListener("click", () => {
    POPUP_SHOWED = false;
    popupOneButton.style.display = "none";
    overlay.classList.remove("show");
    popupOneButton.className = "popup"; // Сброс классов
});

// Инициализация QR сканера
async function initScanner() {
    try {
        // Динамический импорт QR сканера
        const QrScannerModule = await import('./qr-scanner.min.js');
        const QrScanner = QrScannerModule.default;

        const video = document.getElementById('qr-video');

        scanner = new QrScanner(video, result => setResult(result), {
            onDecodeError: error => {
                // Игнорируем ошибки декодирования
            },
            highlightScanRegion: true,
            highlightCodeOutline: true,
        });

        await scanner.start();
        console.log("QR сканер запущен");
    } catch (error) {
        console.error("Ошибка инициализации сканера:", error);
        showPopupOneButton("Ошибка доступа к камере", "Обновить страницу", true);
    }
}

// Запуск сканера при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    initScanner();
});
